package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func TestCanceledUpgradeReturnsInterruptedExitCode(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	app := Init()
	app.ExitErrHandler = func(*cli.Context, error) {}
	err := app.RunContext(ctx, []string{"bizyair", "upgrade", "--check"})

	var exitCoder cli.ExitCoder
	if !errors.As(err, &exitCoder) {
		t.Fatalf("canceled upgrade error type = %T, want cli.ExitCoder", err)
	}
	if got := exitCoder.ExitCode(); got != meta.InterruptedExitCode {
		t.Fatalf("canceled upgrade exit code = %d, want %d", got, meta.InterruptedExitCode)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled upgrade error = %v, want wrapped context.Canceled", err)
	}
}
