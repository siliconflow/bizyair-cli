package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ModelzooList(c *cli.Context) error {
	args := parseArgument(c, meta.CmdLs)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	result := actions.ListModelzooEndpoints(c.Context, client, c.String("keyword"), c.String("billing-unit"), c.String("sort"), c.Bool("show-deprecated"))
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	if len(result.Models) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_empty", nil))
		return nil
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_title", nil))
	printModelzooTable(os.Stdout, result.Models)
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_count", map[string]any{"Count": len(result.Models)}))
	return nil
}

func printModelzooTable(w io.Writer, models []lib.ModelzooModelFlat) {
	table := newTable(w, 5)
	colWs := tableColumnWidths(5, terminalWidth())
	table.Header(
		truncateCell(i18n.T("cli.modelzoo.ls_header_name", nil), colWs[0]-cellPadWidth),
		truncateCell(i18n.T("cli.modelzoo.ls_header_endpoint", nil), colWs[1]-cellPadWidth),
		truncateCell(i18n.T("cli.modelzoo.ls_header_manufacturer", nil), colWs[2]-cellPadWidth),
		truncateCell(i18n.T("cli.modelzoo.ls_header_category", nil), colWs[3]-cellPadWidth),
		truncateCell(i18n.T("cli.modelzoo.ls_header_version", nil), colWs[4]-cellPadWidth),
	)
	for _, m := range models {
		table.Append([]string{
			truncateCell(dashIfEmpty(m.DisplayName), colWs[0]-cellPadWidth),
			truncateCell(m.Endpoint, colWs[1]-cellPadWidth),
			truncateCell(dashIfEmpty(i18n.APITranslate("manufacturer", m.Manufacturer)), colWs[2]-cellPadWidth),
			truncateCell(dashIfEmpty(m.Category), colWs[3]-cellPadWidth),
			truncateCell(dashIfEmpty(i18n.APITranslate("version", m.ModelVersion)), colWs[4]-cellPadWidth),
		})
	}
	table.Render()
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
