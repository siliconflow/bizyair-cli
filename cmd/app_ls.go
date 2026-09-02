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

func AppList(c *cli.Context) error {
	args := parseArgument(c, meta.CmdApp)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	page := c.Int("page")
	if page < 1 {
		page = 1
	}
	pageSize := c.Int("page-size")
	if pageSize < 1 {
		pageSize = 50
	}
	keyword := c.String("keyword")
	sort := c.String("sort")
	baseModel := c.String("base-model")

	result := actions.ListAIApplications(c.Context, client, keyword, sort, nil, page, pageSize)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	apps := lib.FilterAIApplications(result.Apps, lib.AIApplicationFilter{
		Sort:      sort,
		BaseModel: baseModel,
		Keyword:   keyword,
	})
	if len(apps) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.app.ls_empty", nil))
		return nil
	}

	total := result.Total
	if baseModel != "" || keyword != "" {
		total = len(apps)
	}
	if total < len(apps) {
		total = len(apps)
	}
	pages := (total + pageSize - 1) / pageSize
	if pages < 1 {
		pages = 1
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.app.ls_title", nil))
	printAIApplicationTable(os.Stdout, apps)
	fmt.Fprintln(os.Stdout, i18n.T("cli.app.ls_page", map[string]any{
		"Current": page, "Pages": pages, "Total": total, "Shown": len(apps),
	}))
	if page < pages {
		fmt.Fprintln(os.Stdout, i18n.T("cli.app.ls_page_next", map[string]any{"Next": page + 1}))
	}
	return nil
}

func printAIApplicationTable(w io.Writer, apps []*lib.BizyModelInfo) {
	table := newTable(w, 6)
	colWs := tableColumnWidths(6, terminalWidth())
	header := lib.AIApplicationTableColumns()
	table.Header(
		truncateCell(header[0], colWs[0]-cellPadWidth),
		truncateCell(header[1], colWs[1]-cellPadWidth),
		truncateCell(header[2], colWs[2]-cellPadWidth),
		truncateCell(header[3], colWs[3]-cellPadWidth),
		truncateCell(header[4], colWs[4]-cellPadWidth),
		truncateCell(header[5], colWs[5]-cellPadWidth),
	)
	for _, app := range apps {
		cells := lib.AIApplicationTableRow(app)
		table.Append([]string{
			truncateCell(cells[0], colWs[0]-cellPadWidth),
			truncateCell(cells[1], colWs[1]-cellPadWidth),
			truncateCell(cells[2], colWs[2]-cellPadWidth),
			truncateCell(cells[3], colWs[3]-cellPadWidth),
			truncateCell(cells[4], colWs[4]-cellPadWidth),
			truncateCell(cells[5], colWs[5]-cellPadWidth),
		})
	}
	table.Render()
}