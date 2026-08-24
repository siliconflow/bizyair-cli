package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// PublicModel 切换指定模型版本的公开状态。通过 --version-ids 直接传入版本 ID
// 列表（逗号分隔），与 TUI 的 P 键行为对齐。必须显式给出 --public（true/false），
// 未提供 -y 时需交互确认。
func PublicModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdPublic)
	setLogVerbose(args.Verbose)
	logArguments(args)

	if !c.IsSet("public") {
		return cli.Exit(i18n.NewError("cli.error.flag_value_required", map[string]any{"Flag": "--public"}, nil), meta.LoadError)
	}
	public := c.Bool("public")

	versionIDsStr := c.String("version-ids")
	if strings.TrimSpace(versionIDsStr) == "" {
		return cli.Exit(i18n.NewError("cli.public.version_ids_required", nil, nil), meta.LoadError)
	}
	versionIDs, err := parseVersionIDs(versionIDsStr)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	if !c.Bool("yes") {
		fmt.Fprint(os.Stdout, i18n.T("cli.public.confirm", map[string]any{"Count": len(versionIDs), "Public": public}))
		var resp string
		_, _ = fmt.Scanln(&resp)
		if !confirmYes(resp) {
			return cli.Exit(i18n.NewError("cli.public.canceled", nil, nil), meta.LoadError)
		}
	}

	apiKey := args.ApiKey
	if apiKey == "" {
		var e error
		apiKey, e = lib.NewSfFolder().GetKey()
		if e != nil {
			return cli.Exit(e, meta.LoadError)
		}
	}
	client := lib.NewClient(args.BaseDomain, apiKey)

	if err := actions.ToggleModelPublicContext(c.Context, client, versionIDs, public); err != nil {
		return cli.Exit(err, meta.ServerError)
	}
	fmt.Fprintln(os.Stdout, i18n.T("cli.public.success"))
	return nil
}
