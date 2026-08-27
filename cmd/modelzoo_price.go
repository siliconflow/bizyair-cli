package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mattn/go-runewidth"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
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

	pt := result.PriceTable
	if pt == nil || len(pt.Columns) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_empty", nil))
		return nil
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_title", map[string]any{"Endpoint": endpoint}))

	colW := make([]int, len(pt.Columns))
	headers := make([]string, len(pt.Columns))
	for i, col := range pt.Columns {
		title := col.FieldLabel
		if title == "" {
			title = col.FieldName
		}
		headers[i] = title
		colW[i] = runewidth.StringWidth(title)
	}
	fmt.Fprintln(os.Stdout, joinCells(headers, colW))

	for _, row := range pt.Cells {
		cells := make([]string, len(pt.Columns))
		for i := range pt.Columns {
			if i < len(row) {
				cells[i] = formatPriceCell(row[i])
			} else {
				cells[i] = "-"
			}
			if w := runewidth.StringWidth(cells[i]); w > colW[i] {
				colW[i] = w
			}
		}
		fmt.Fprintln(os.Stdout, joinCells(cells, colW))
	}
	return nil
}

func formatPriceCell(cell lib.PriceTableCell) string {
	if cell.ValueStr != "" {
		return cell.ValueStr
	}
	if cell.Amount > 0 {
		s := strconv.FormatFloat(cell.Amount, 'f', 2, 64)
		if cell.UnitName != "" {
			return fmt.Sprintf("%s %s", s, cell.UnitName)
		}
		return s
	}
	return "-"
}
