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

// RemoveModel 删除模型。支持两种定位方式：
//   - 位置参数 <id>：直接按数字 ID 删除；
//   - -n <name> [-t <type>]：按名称（可选限定类型）解析后删除。
//
// 未提供 -y 时需交互确认，避免误删。
func RemoveModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdRm)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	var modelID int64
	idStr := firstModelArg(c.Args())
	if idStr == "" {
		if v, ok := tailFlagValue(c.Args(), "n", "name"); ok {
			args.Name = v
		}
	}
	if modelType, ok := tailFlagValue(c.Args(), "t", "type"); ok {
		args.Type = modelType
	}
	if idStr != "" {
		mid, err := resolveModelID(c, client, args, idStr)
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		modelID = mid
	} else {
		// 按名称删除时必须先校验类型（空类型会返回“模型类型不能为空”）。
		if err := lib.ValidateModelType(args.Type); err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		if args.Name == "" {
			return cli.Exit(i18n.NewError("cli.rm.name_or_id_required", nil, nil), meta.LoadError)
		}
		mid, err := resolveModelID(c, client, args, args.Name)
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		modelID = mid
	}

	if !c.Bool("yes") && !tailYesFlag(c.Args()) {
		fmt.Fprint(os.Stdout, i18n.T("cli.model.rm_confirm", map[string]any{"ID": modelID}))
		var resp string
		_, _ = fmt.Scanln(&resp)
		if !confirmYes(resp) {
			return cli.Exit(i18n.NewError("cli.model.rm_canceled", nil, nil), meta.LoadError)
		}
	}

	result := actions.DeleteModelContext(c.Context, client, modelID)
	if !result.Success {
		return cli.Exit(result.Error, meta.ServerError)
	}
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.remove_success"))
	return nil
}
