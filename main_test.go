package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/siliconflow/bizyair-cli/meta"
	urfavecli "github.com/urfave/cli/v2"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: 0},
		{name: "ordinary error", err: errors.New("failed"), want: 1},
		{name: "explicit exit code", err: urfavecli.Exit("failed", meta.ServerError), want: meta.ServerError},
		{name: "zero exit code with error", err: urfavecli.Exit("failed", 0), want: 1},
		{name: "canceled", err: context.Canceled, want: meta.InterruptedExitCode},
		{
			name: "wrapped canceled takes precedence over explicit code",
			err:  urfavecli.Exit(fmt.Errorf("request failed: %w", context.Canceled), meta.ServerError),
			want: meta.InterruptedExitCode,
		},
		{name: "deadline is not user interruption", err: context.DeadlineExceeded, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Fatalf("exitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestPreserveContextCancellation(t *testing.T) {
	original := errors.New("command failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := preserveContextCancellation(ctx, original)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("preserved error = %v, want context.Canceled", err)
	}
	if !errors.Is(err, original) {
		t.Fatalf("preserved error = %v, want original error", err)
	}

	if got := preserveContextCancellation(context.Background(), original); got != original {
		t.Fatalf("live context changed error to %v", got)
	}
	if !errors.Is(preserveContextCancellation(ctx, nil), context.Canceled) {
		t.Fatal("nil command error did not preserve canceled context")
	}
}

func TestRunContextCancellationTakesPrecedenceOverArgumentError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runContext(ctx, []string{"bizyair", "--lang", "unsupported"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runContext() error = %v, want context.Canceled", err)
	}
	if got := exitCode(err); got != meta.InterruptedExitCode {
		t.Fatalf("exitCode(runContext()) = %d, want %d", got, meta.InterruptedExitCode)
	}
}
