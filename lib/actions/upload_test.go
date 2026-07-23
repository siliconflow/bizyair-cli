package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

type cancellationUploadAPI struct {
	lib.BizyAPI
	checkErr  error
	ossSign   func(context.Context) (*lib.Response[lib.FilesResp], error)
	commitErr error
}

func (a *cancellationUploadAPI) CheckModelExistsContext(context.Context, string, string) (bool, error) {
	return false, a.checkErr
}

func (a *cancellationUploadAPI) OssSignContext(ctx context.Context, _, _ string) (*lib.Response[lib.FilesResp], error) {
	if a.ossSign != nil {
		return a.ossSign(ctx)
	}
	return &lib.Response[lib.FilesResp]{
		Data: lib.FilesResp{
			File: &lib.FileInfo{Id: 1, ObjectKey: "existing/model"},
		},
	}, nil
}

func (a *cancellationUploadAPI) CommitModelV2Context(
	context.Context,
	string,
	string,
	[]*lib.ModelVersion,
) (*lib.Response[lib.ModelCommitResp], error) {
	if a.commitErr != nil {
		return nil, a.commitErr
	}
	return &lib.Response[lib.ModelCommitResp]{}, nil
}

func TestExecuteUploadCancellationStages(t *testing.T) {
	t.Run("before validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result := ExecuteUpload(nil, UploadInput{Context: ctx}, nil)
		assertCanceledUpload(t, result)
	})

	t.Run("model existence check", func(t *testing.T) {
		input := validUploadInput(t, context.Background())
		input.Overwrite = false
		api := &cancellationUploadAPI{checkErr: fmt.Errorf("check failed: %w", context.Canceled)}

		result := ExecuteUpload(api, input, nil)
		assertCanceledUpload(t, result)
	})

	t.Run("file upload", func(t *testing.T) {
		input := validUploadInput(t, context.Background())
		api := &cancellationUploadAPI{
			ossSign: func(context.Context) (*lib.Response[lib.FilesResp], error) {
				return nil, fmt.Errorf("signature failed: %w", context.Canceled)
			},
		}

		result := ExecuteUpload(api, input, nil)
		assertCanceledUpload(t, result)
	})

	t.Run("model commit", func(t *testing.T) {
		input := validUploadInput(t, context.Background())
		api := &cancellationUploadAPI{
			commitErr: fmt.Errorf("commit failed: %w", context.Canceled),
		}

		result := ExecuteUpload(api, input, nil)
		assertCanceledUpload(t, result)
		if result.SuccessCount != 1 {
			t.Fatalf("SuccessCount = %d, want completed file count 1", result.SuccessCount)
		}
	})
}

func TestExecuteUploadDeadlineIsNotUserCancellation(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	input := validUploadInput(t, ctx)
	result := ExecuteUpload(&cancellationUploadAPI{}, input, nil)
	if result.CanceledByUser {
		t.Fatal("deadline exceeded was classified as a user cancellation")
	}
	if len(result.Errors) == 0 || !errors.Is(result.Errors[0], context.DeadlineExceeded) {
		t.Fatalf("errors = %v, want context.DeadlineExceeded", result.Errors)
	}
}

func TestUploadVersionsStopsQueuedWorkOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := validUploadInput(t, ctx)
	input.Versions = []VersionInput{
		input.Versions[0],
		input.Versions[0],
		input.Versions[0],
		input.Versions[0],
	}

	started := make(chan struct{}, len(input.Versions))
	api := &cancellationUploadAPI{
		ossSign: func(ctx context.Context) (*lib.Response[lib.FilesResp], error) {
			started <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	resultCh := make(chan UploadResult, 1)
	go func() {
		resultCh <- uploadVersionsConcurrently(ctx, api, input, nil)
	}()

	for range 3 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for active upload workers")
		}
	}
	cancel()

	var result UploadResult
	select {
	case result = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for canceled upload")
	}
	assertCanceledUpload(t, result)

	if queuedStarts := len(started); queuedStarts != 0 {
		t.Fatalf("%d queued upload(s) started after cancellation", queuedStarts)
	}
}

func validUploadInput(t *testing.T, ctx context.Context) UploadInput {
	t.Helper()

	path := filepath.Join(t.TempDir(), "model.safetensors")
	if err := os.WriteFile(path, []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}

	return UploadInput{
		Context:   ctx,
		ApiKey:    "api-key",
		ModelType: string(meta.TypeCheckpoint),
		ModelName: "test-model",
		Overwrite: true,
		Versions: []VersionInput{{
			Version:      "v1.0",
			Path:         path,
			Introduction: "introduction",
			CoverUrl:     " ",
		}},
	}
}

func assertCanceledUpload(t *testing.T, result UploadResult) {
	t.Helper()
	if result.Success || !result.CanceledByUser {
		t.Fatalf("result = %#v, want canceled upload", result)
	}
}
