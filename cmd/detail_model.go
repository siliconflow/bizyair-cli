package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// DetailModel 按模型 ID 或名称查询并打印模型详情（基本信息、统计与版本列表）。
// 位置参数 <id|name>：数字按 ID 处理，否则按名称解析。-t 可在按名解析时限定类型。
func DetailModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdDetail)
	setLogVerbose(args.Verbose)
	logArguments(args)

	idStr := firstModelArg(c.Args())
	if idStr == "" {
		return cli.Exit(i18n.NewError("cli.detail.id_required", nil, nil), meta.LoadError)
	}

	if modelType, ok := tailFlagValue(c.Args(), "t", "type"); ok {
		args.Type = modelType
	}
	if _, err := strconv.ParseInt(idStr, 10, 64); err != nil {
		if args.Type != "" {
			if verr := lib.ValidateModelType(args.Type); verr != nil {
				return cli.Exit(verr, meta.LoadError)
			}
		}
	}

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	modelID, err := resolveModelID(c, client, args, idStr)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	result := actions.GetModelDetailContext(c.Context, client, modelID)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	detail := result.Detail
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_title", nil))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_id", map[string]any{"ID": detail.Id}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_name", map[string]any{"Name": detail.Name}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_type", map[string]any{"Type": detail.Type}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_created", map[string]any{"Date": formatTimestampCLI(detail.CreatedAt)}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_updated", map[string]any{"Date": formatTimestampCLI(detail.UpdatedAt)}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_versions", map[string]any{"Count": len(detail.Versions)}))

	if len(detail.Versions) > 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_versions_header", nil))
		for _, v := range detail.Versions {
			fmt.Fprintln(os.Stdout, i18n.T("cli.model.detail_version", map[string]any{
				"Version":   v.Version,
				"Size":      format.FormatBytes(v.FileSize),
				"Status":    versionStatusText(v.Available),
				"BaseModel": v.BaseModel,
				"Public":    v.Public,
			}))
		}
	}
	return nil
}

// versionStatusText 将版本可用状态映射为简短文本。
func versionStatusText(available bool) string {
	if available {
		return i18n.T("cli.model.detail_status_available", nil)
	}
	return i18n.T("cli.model.detail_status_unavailable", nil)
}
