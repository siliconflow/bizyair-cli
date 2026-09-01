package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

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

	page := c.Int("page")
	if page < 1 {
		page = 1
	}
	pageSize := c.Int("page-size")
	if pageSize < 1 {
		pageSize = 50
	}
	result := actions.ListModelzooEndpoints(c.Context, client, c.String("keyword"), "", c.String("sort"), c.Bool("show-deprecated"), page, pageSize)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	// 允许不引号输入多词能力值，例如 `--cap text to video` 会被 shell 拆分为
	// --cap(text) + 位置参数(to video)，这里把紧随其后的位置参数拼回能力值。
	// 遇到下一个 flag（- 开头）即停止，保证后续其它 flag 正常生效。
	capability := c.String("capability")
	if capability != "" {
		var extra []string
		for _, arg := range c.Args().Slice() {
			if strings.HasPrefix(arg, "-") {
				break
			}
			extra = append(extra, arg)
		}
		if len(extra) > 0 {
			capability = strings.TrimSpace(capability + " " + strings.Join(extra, " "))
		}
	}
	manufacturer := c.String("manufacturer")
	series := c.String("series")
	version := c.String("version")

	models := lib.FilterModelzooModels(result.Models, lib.ModelzooFilter{
		Capability: capability, Manufacturer: manufacturer, Series: series, Version: version,
	})
	if len(models) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_empty", nil))
		return nil
	}

	total := result.Total
	if capability != "" || manufacturer != "" || series != "" || version != "" {
		total = len(models)
	}
	if total < len(models) {
		total = len(models)
	}
	pages := (total + pageSize - 1) / pageSize
	if pages < 1 {
		pages = 1
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_title", nil))
	printModelzooTable(os.Stdout, models)
	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_page", map[string]any{
		"Current": page, "Pages": pages, "Total": total, "Shown": len(models),
	}))
	if page < pages {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.ls_page_next", map[string]any{"Next": page + 1}))
	}
	return nil
}

func printModelzooTable(w io.Writer, models []lib.ModelzooModelFlat) {
	table := newTable(w, 5)
	colWs := tableColumnWidths(5, terminalWidth())
	header := lib.ModelzooTableColumns()
	table.Header(
		truncateCell(header[0], colWs[0]-cellPadWidth),
		truncateCell(header[1], colWs[1]-cellPadWidth),
		truncateCell(header[2], colWs[2]-cellPadWidth),
		truncateCell(header[3], colWs[3]-cellPadWidth),
		truncateCell(header[4], colWs[4]-cellPadWidth),
	)
	for _, m := range models {
		cells := lib.ModelzooTableRow(m)
		table.Append([]string{
			truncateCell(cells[0], colWs[0]-cellPadWidth),
			truncateCell(cells[1], colWs[1]-cellPadWidth),
			truncateCell(cells[2], colWs[2]-cellPadWidth),
			truncateCell(cells[3], colWs[3]-cellPadWidth),
			truncateCell(cells[4], colWs[4]-cellPadWidth),
		})
	}
	table.Render()
}
