package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/urfave/cli/v2"
)

func TestCanceledUploadReturnsInterruptedExitCode(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	err := canceledUploadExitError()
	var exitCoder cli.ExitCoder
	if !errors.As(err, &exitCoder) {
		t.Fatalf("canceled upload error type = %T, want cli.ExitCoder", err)
	}
	if got := exitCoder.ExitCode(); got != interruptedExitCode {
		t.Fatalf("canceled upload exit code = %d, want %d", got, interruptedExitCode)
	}
}

func TestUploadYamlModelsStopsAfterCancellation(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	models := []config.YamlModel{
		{Name: "first", Type: "Checkpoint"},
		{Name: "second", Type: "Checkpoint"},
	}
	calls := 0
	results, err := uploadYamlModels(
		ctx,
		models,
		config.NewArgument(),
		"api-key",
		func(
			context.Context,
			string,
			string,
			string,
			string,
			[]config.YamlVersion,
			bool,
		) modelUploadResult {
			calls++
			cancel()
			return modelUploadResult{
				ModelName: "first",
				ModelType: "Checkpoint",
				Success:   true,
			}
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("uploadYamlModels() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("upload model calls = %d, want 1", calls)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("partial results = %#v, want the completed first model only", results)
	}
}

func TestCanceledYamlUploadReturnsInterruptedExitCode(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	app := Init()
	app.ExitErrHandler = func(*cli.Context, error) {}
	err := app.RunContext(ctx, []string{"bizyair", "upload", "--file", "not-read-after-cancel.yaml"})

	var exitCoder cli.ExitCoder
	if !errors.As(err, &exitCoder) {
		t.Fatalf("canceled YAML upload error type = %T, want cli.ExitCoder", err)
	}
	if got := exitCoder.ExitCode(); got != interruptedExitCode {
		t.Fatalf("canceled YAML upload exit code = %d, want %d", got, interruptedExitCode)
	}
}
