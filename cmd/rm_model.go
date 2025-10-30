package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func RemoveModel(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdRm)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	// 获取API Key
	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 先查找模型以获取ID
	listInput := actions.ListModelsInput{
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Keyword:    args.Name,
	}
	listResult := actions.ListModels(listInput)
	if listResult.Error != nil {
		return cli.Exit(listResult.Error, meta.ServerError)
	}

	// 查找匹配的模型
	var modelId int64
	for _, model := range listResult.Models {
		if model.Name == args.Name {
			modelId = model.Id
			break
		}
	}

	if modelId == 0 {
		return cli.Exit(fmt.Errorf("model '%s' not found", args.Name), meta.LoadError)
	}

	// 调用统一的删除逻辑
	result := actions.DeleteModel(apiKey, args.BaseDomain, modelId)
	if !result.Success {
		return cli.Exit(result.Error, meta.ServerError)
	}

	fmt.Fprintln(os.Stdout, "Model removed successfully.")
	return nil
}
