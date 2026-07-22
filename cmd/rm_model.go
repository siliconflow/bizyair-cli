package cmd

import (
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func RemoveModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdRm)
	setLogVerbose(args.Verbose)
	logArguments(args)

	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	// 获取API Key
	apiKey := args.ApiKey
	if apiKey == "" {
		var err error
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 先查找模型以获取ID
	client := lib.NewClient(args.BaseDomain, apiKey)
	listInput := actions.ListModelsInput{
		Context:    c.Context,
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Keyword:    args.Name,
	}
	listResult := actions.ListModels(client, listInput)
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
		return cli.Exit(i18n.NewError("error.model.not_found_named", map[string]any{"Name": args.Name}, nil), meta.LoadError)
	}

	// 调用统一的删除逻辑
	result := actions.DeleteModelContext(c.Context, client, modelId)
	if !result.Success {
		return cli.Exit(result.Error, meta.ServerError)
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.model.remove_success"))
	return nil
}
