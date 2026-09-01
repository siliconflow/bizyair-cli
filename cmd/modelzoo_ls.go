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

	models := filterModelzooModels(result.Models, capability, manufacturer, series, version)
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

// filterModelzooModels 按能力/厂商/系列/版本条件过滤模型列表，逻辑与 TUI 保持一致。
func filterModelzooModels(models []lib.ModelzooModelFlat, capability, manufacturer, series, version string) []lib.ModelzooModelFlat {
	if capability == "" && manufacturer == "" && series == "" && version == "" {
		return models
	}
	keep := make([]lib.ModelzooModelFlat, 0, len(models))
	for _, m := range models {
		if capability != "" && !hasModelzooCapability(m, capability) {
			continue
		}
		if manufacturer != "" && !matchFilterValue(m.Manufacturer, "manufacturer", manufacturer) {
			continue
		}
		if series != "" {
			s := m.Series
			if s == "" {
				s = lib.SeriesFromEndpoint(m.Endpoint)
			}
			if s != "" && !matchFilterValue(s, "series", series) {
				continue
			}
		}
		if version != "" && !matchFilterValue(m.ModelVersion, "version", version) {
			continue
		}
		keep = append(keep, m)
	}
	return keep
}

func hasModelzooCapability(m lib.ModelzooModelFlat, capability string) bool {
	for _, t := range m.Tags {
		if matchFilterValue(t, "tag", capability) {
			return true
		}
	}
	return matchFilterValue(m.Category, "category", capability)
}

// matchFilterValue 匹配原始值或其在当前语言下的展示值，
// 例如厂商原始值 "117" 展示为 "ByteDance"，两者都应可通过 --manufacturer 命中。
func matchFilterValue(raw, domain, want string) bool {
	if raw == "" || want == "" {
		return false
	}
	if strings.EqualFold(raw, want) {
		return true
	}
	return strings.EqualFold(i18n.APITranslate(domain, raw), want)
}
