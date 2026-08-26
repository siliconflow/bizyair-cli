package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// PublicModel 切换模型版本的公开状态。通过位置参数 <id|name> 定位模型，
// 自动获取该模型所有版本 ID 并切换公开状态。
// 不提供 --public 时自动切换（检测当前状态后翻转）。
func PublicModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdPublic)
	setLogVerbose(args.Verbose)
	logArguments(args)

	idStr := c.Args().First()
	if idStr == "" {
		idStr = args.Name
	}
	if idStr == "" {
		return cli.Exit(i18n.NewError("cli.public.id_or_name_required", nil, nil), meta.LoadError)
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

	detailResult := actions.GetModelDetailContext(c.Context, client, modelID)
	if detailResult.Error != nil {
		return cli.Exit(detailResult.Error, meta.ServerError)
	}
	detail := detailResult.Detail
	if detail == nil || len(detail.Versions) == 0 {
		return cli.Exit(i18n.NewError("cli.public.no_versions", nil, nil), meta.LoadError)
	}

	versionIDs := actions.ExtractVersionIDs(detail)

	var public bool
	if c.IsSet("public") {
		publicStr := strings.ToLower(strings.TrimSpace(c.String("public")))
		public = publicStr == "true" || publicStr == "1" || publicStr == "yes"
	} else {
		public = !detail.Versions[0].Public
	}

	if !c.Bool("yes") {
		fmt.Fprint(os.Stdout, i18n.T("cli.public.confirm", map[string]any{"Count": len(versionIDs), "Public": public}))
		var resp string
		_, _ = fmt.Scanln(&resp)
		if !confirmYes(resp) {
			return cli.Exit(i18n.NewError("cli.public.canceled", nil, nil), meta.LoadError)
		}
	}

	if err := actions.ToggleModelPublicContext(c.Context, client, versionIDs, public); err != nil {
		return cli.Exit(err, meta.ServerError)
	}
	fmt.Fprintln(os.Stdout, i18n.T("cli.public.success"))
	return nil
}
