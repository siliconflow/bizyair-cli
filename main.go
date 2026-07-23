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

func runContext(ctx context.Context, args []string) error {
	defer func() {
		logs.Flush()
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	programName := "bizyair"
	languageArgs := args
	if len(args) > 0 {
		programName = args[0]
		languageArgs = args[1:]
	}
	language, commandArgs, err := i18n.ResolveArgs(languageArgs, os.LookupEnv)
	if err != nil {
		return err
	}
	if err := i18n.Configure(language); err != nil {
		return err
	}

	app := cmd.Init()
	// Keep process termination in main so every returned error, including
	// localized usage errors, receives a reliable non-zero exit code.
	app.ExitErrHandler = func(*urfavecli.Context, error) {}
	return app.RunContext(ctx, append([]string{programName}, commandArgs...))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitCoder urfavecli.ExitCoder
	if errors.As(err, &exitCoder) && exitCoder.ExitCode() != 0 {
		return exitCoder.ExitCode()
	}
	return 1
}
