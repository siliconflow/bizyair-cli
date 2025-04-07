package main

import (
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/siliconcloud-cli/cmd"
	"os"
)

func main() {
	Run()
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
