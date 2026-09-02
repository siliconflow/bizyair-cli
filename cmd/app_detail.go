package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func AppDetail(c *cli.Context) error {
	args := parseArgument(c, meta.CmdApp)
	setLogVerbose(args.Verbose)
	logArguments(args)

	idStr := c.Args().First()
	if idStr == "" {
		return cli.Exit(i18n.NewError("cli.app.id_required", nil, nil), meta.LoadError)
	}
	appID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return cli.Exit(i18n.NewError("cli.app.not_found", map[string]any{"ID": idStr}, err), meta.LoadError)
	}

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	result := actions.GetWebAppDetail(c.Context, client, appID)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}
	d := result.Detail
	if d == nil {
		return cli.Exit(i18n.NewError("cli.app.not_found", map[string]any{"ID": idStr}, nil), meta.ServerError)
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_title", nil))
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_id", map[string]any{"ID": d.Id}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_name", map[string]any{"Name": d.Name}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_base_model", map[string]any{"Model": lib.DisplayOrDash(d.BaseModel)}))
	owner := d.NickName
	if owner == "" {
		owner = d.Creator
	}
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_owner", map[string]any{"Owner": lib.DisplayOrDash(owner)}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_created", map[string]any{"Created": formatTimestampCLI(d.CreatedAt)}))
	if d.Description != "" {
		fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_description", map[string]any{"Desc": d.Description}))
	}

	if len(d.InputNodes) > 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_params_header", nil))
		for _, n := range d.InputNodes {
			fmt.Fprintln(os.Stdout, i18n.T("cli.app.detail_param_line", map[string]any{
				"Name":  lib.WebAppNodeParamKey(n),
				"Label": lib.WebAppFieldLabel(n),
				"Type":  variableTypeDisplayName(lib.WebAppNodeVariableType(n)),
			}))
		}
	}
	return nil
}