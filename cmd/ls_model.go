package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ListModel(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdLs)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 打开浏览器查看"我的模型"
	msg, err := lib.OpenBrowser(lib.MyModelsURL)
	if err != nil {
		// 如果无法打开浏览器，提示用户手动访问
		fmt.Fprintf(os.Stderr, "无法打开浏览器: %v\n", err)
		fmt.Fprintf(os.Stdout, "请在浏览器中访问: %s\n", lib.MyModelsURL)
		return cli.Exit(err, meta.LoadError)
	}

	// 成功打开浏览器
	fmt.Fprintln(os.Stdout, msg)
	fmt.Fprintf(os.Stdout, "访问: %s\n", lib.MyModelsURL)
	return nil
}
