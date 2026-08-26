package cmd

import (
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/urfave/cli/v2"
)

// confirmYes 判断用户输入是否表示确认（以 y/Y 开头）。空输入视为否。
func confirmYes(response string) bool {
	response = strings.TrimSpace(response)
	if response == "" {
		return false
	}
	return strings.EqualFold(response[:1], "y")
}

// resolveModelID 将位置参数或名称解析为模型 ID：
//   - 纯数字按模型 ID 处理；
//   - 否则按名称（可选以 args.Type 限定类型）调用列表接口精确匹配。
func resolveModelID(c *cli.Context, client *lib.Client, args *config.Argument, idStr string) (int64, error) {
	if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
		return id, nil
	}

	listInput := actions.ListModelsInput{
		Context:    c.Context,
		ApiKey:     args.ApiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Keyword:    idStr,
	}
	listResult := actions.ListModels(client, listInput)
	if listResult.Error != nil {
		return 0, listResult.Error
	}
	for _, m := range listResult.Models {
		if m.Name == idStr {
			return m.Id, nil
		}
	}
	return 0, i18n.NewError("cli.detail.not_found", map[string]any{"ID": idStr}, nil)
}
