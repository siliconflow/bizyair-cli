package cmd

import (
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ModelzooDetail(c *cli.Context) error {
	args := parseArgument(c, meta.CmdDetail)
	setLogVerbose(args.Verbose)
	logArguments(args)

	endpoint := c.Args().First()
	if endpoint == "" {
		return cli.Exit(i18n.NewError("cli.modelzoo.endpoint_required", nil, nil), meta.LoadError)
	}

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	result := actions.GetEndpointDetail(c.Context, client, endpoint)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	d := result.Detail
	if d == nil {
		return cli.Exit(i18n.NewError("cli.modelzoo.detail_not_found", map[string]any{"Endpoint": endpoint}, nil), meta.ServerError)
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_title", nil))
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_endpoint", map[string]any{"Endpoint": d.Endpoint}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_name", map[string]any{"Name": d.DisplayName}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_manufacturer", map[string]any{"Manufacturer": i18n.APITranslate("manufacturer", d.Manufacturer)}))
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_category", map[string]any{"Category": d.Category}))
	if d.BillingUnit != "" {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_billing_unit", map[string]any{"Unit": d.BillingUnit}))
	}
	if d.MinCredits > 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_min_credits", map[string]any{"Cost": formatCredits(d.MinCredits)}))
	}
	if d.Description != "" {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_description", map[string]any{"Desc": d.Description}))
	}

	if len(d.InputParams) > 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_params_header", nil))
		for _, p := range d.InputParams {
			req := ""
			if p.Required {
				req = i18n.T("cli.modelzoo.param_required", nil)
			}
			fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.detail_param_line", map[string]any{
				"Name":     p.ParamKey(),
				"Label":    p.FieldLabel,
				"Type":     p.VariableType,
				"Required": req,
			}))
		}
	}
	return nil
}
