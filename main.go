package main

import (
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/cmd"
)

func main() {
	Run()
	// test()
}

func Run() {
	defer func() {
		logs.Flush()
	}()

	cli := cmd.Init()
	err := cli.Run(os.Args)
	if err != nil {
		logs.Errorf("%v\n", err)
	}
}

