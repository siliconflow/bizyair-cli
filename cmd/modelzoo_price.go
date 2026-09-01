package cmd

import (
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ModelzooPrice(c *cli.Context) error {
	args := parseArgument(c, meta.CmdPrice)
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

	result := actions.GetPriceTable(c.Context, client, endpoint)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	if len(result.PriceTables) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_empty", nil))
		return nil
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_title", map[string]any{"Endpoint": endpoint}))
	for _, pt := range result.PriceTables {
		table := newTable(os.Stdout, len(format.PriceTableColumns(pt, true))+2)
		cols := make([]any, 0, 5)
		for _, c := range format.PriceTableColumns(pt, true) {
			cols = append(cols, c)
		}
		table.Header(cols...)
		for _, row := range format.PriceTableRows(pt, true) {
			table.Append(row)
		}
		table.Render()
	}
	return nil
}
