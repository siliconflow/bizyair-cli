package main

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/cmd"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

func main() {
	if err := Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func Run() error {
	defer func() {
		logs.Flush()
	}()

	language, err := i18n.Resolve(os.Args[1:], os.LookupEnv)
	if err != nil {
		return err
	}
	if err := i18n.Configure(language); err != nil {
		return err
	}

	cli := cmd.Init()
	return cli.Run(os.Args)
}
