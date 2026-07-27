package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
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
	if got := exitCoder.ExitCode(); got != meta.InterruptedExitCode {
		t.Fatalf("canceled upload exit code = %d, want %d", got, meta.InterruptedExitCode)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled upload error = %v, want wrapped context.Canceled", err)
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

func TestParseVersionPublic(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    []bool
		wantErr bool
	}{
		{name: "empty", values: nil, want: []bool{}},
		{name: "standard values", values: []string{"true", "false"}, want: []bool{true, false}},
		{name: "strconv compatible values", values: []string{"TRUE", "0", "1"}, want: []bool{true, false, true}},
		{name: "invalid", values: []string{"maybe"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVersionPublic(tt.values)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseVersionPublic(%v) error = %v, wantErr %v", tt.values, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseVersionPublic(%v) = %v, want %v", tt.values, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseVersionPublic(%v) = %v, want %v", tt.values, got, tt.want)
				}
			}
		})
	}
}

func TestInvalidPublicValueFailsBeforeUploadSetup(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	app := Init()
	app.ExitErrHandler = func(*cli.Context, error) {}
	err := app.Run([]string{"bizyair", "upload", "--pub", "maybe"})
	if err == nil {
		t.Fatal("invalid --pub value unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "invalid public value") {
		t.Fatalf("invalid --pub error = %q", err)
	}
}

func TestOverwriteFlagIsNotAccepted(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	app := Init()
	app.ExitErrHandler = func(*cli.Context, error) {}
	err := app.Run([]string{"bizyair", "upload", "--overwrite"})
	if err == nil {
		t.Fatal("removed --overwrite flag unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "overwrite") {
		t.Fatalf("removed --overwrite error = %q", err)
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
	if got := exitCoder.ExitCode(); got != meta.InterruptedExitCode {
		t.Fatalf("canceled YAML upload exit code = %d, want %d", got, meta.InterruptedExitCode)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled YAML upload error = %v, want wrapped context.Canceled", err)
	}
}
