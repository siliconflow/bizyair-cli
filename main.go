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
	urfavecli "github.com/urfave/cli/v2"
)

func main() {
	if err := Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
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
	languageArgs := args
	if len(languageArgs) > 0 {
		languageArgs = languageArgs[1:]
	}
	language, err := i18n.Resolve(languageArgs, os.LookupEnv)
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
	return app.RunContext(ctx, args)
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
