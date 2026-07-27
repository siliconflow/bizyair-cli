package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/cmd"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	urfavecli "github.com/urfave/cli/v2"
)

func main() {
	if err := Run(); err != nil {
		fmt.Fprintln(os.Stderr, lib.FormatError(err, cmd.Verbose()))
		os.Exit(exitCode(err))
	}
}

func Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runContext(ctx, os.Args)
}

func runContext(ctx context.Context, args []string) (err error) {
	defer func() {
		logs.Flush()
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		err = preserveContextCancellation(ctx, err)
	}()

	programName := "bizyair"
	languageArgs := args
	if len(args) > 0 {
		programName = args[0]
		languageArgs = args[1:]
	}
	language, commandArgs, resolveErr := i18n.ResolveArgs(languageArgs, os.LookupEnv)
	if err := i18n.Configure(language); err != nil {
		return err
	}
	if resolveErr != nil {
		return resolveErr
	}

	app := cmd.Init()
	// Keep process termination in main so every returned error, including
	// localized usage errors, receives a reliable non-zero exit code.
	app.ExitErrHandler = func(*urfavecli.Context, error) {}
	return app.RunContext(ctx, append([]string{programName}, commandArgs...))
}

func preserveContextCancellation(ctx context.Context, err error) error {
	if ctx == nil || !errors.Is(ctx.Err(), context.Canceled) {
		return err
	}
	if err == nil {
		return context.Canceled
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return errors.Join(err, context.Canceled)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) {
		return meta.InterruptedExitCode
	}
	var exitCoder urfavecli.ExitCoder
	if errors.As(err, &exitCoder) && exitCoder.ExitCode() != 0 {
		return exitCoder.ExitCode()
	}
	return 1
}
