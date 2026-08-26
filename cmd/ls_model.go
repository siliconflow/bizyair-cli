package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// ListModel 列出当前用户的模型。默认打印终端表格；传入 --open 时改为在浏览器
// 中打开"我的模型"页面（保留旧行为）。
func ListModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdLs)
	setLogVerbose(args.Verbose)
	logArguments(args)

	if c.Bool("open") {
		endpoints, resolveErr := lib.ResolveServiceEndpoints(args.BaseDomain)
		if resolveErr != nil {
			return cli.Exit(resolveErr, meta.LoadError)
		}
		myModelsURL := endpoints.MyModelsURL()
		msg, err := lib.OpenBrowser(myModelsURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.model.browser_failed", map[string]any{"Cause": err}))
			fmt.Fprintln(os.Stdout, i18n.T("cli.model.visit_manually", map[string]any{"URL": myModelsURL}))
			return cli.Exit(err, meta.LoadError)
		}
		fmt.Fprintln(os.Stdout, msg)
		fmt.Fprintln(os.Stdout, i18n.T("cli.model.visit", map[string]any{"URL": myModelsURL}))
		return nil
	}

	apiKey, client, err := ResolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	listInput := actions.ListModelsInput{
		Context:    c.Context,
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Keyword:    c.String("keyword"),
		Sort:       c.String("sort"),
		Current:    c.Int("page"),
		PageSize:   c.Int("page-size"),
	}
	if bm := c.String("base-model"); bm != "" {
		listInput.BaseModels = []string{bm}
	}
	listResult := actions.ListModels(client, listInput)
	if listResult.Error != nil {
		return cli.Exit(listResult.Error, meta.ServerError)
	}

	models := listResult.Models
	if len(models) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.model.ls_empty"))
		return nil
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.model.ls_title"))
	printModelTable(os.Stdout, models)
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.ls_count", map[string]any{"Count": len(models)}))
	return nil
}

func printModelTable(w io.Writer, models []*lib.BizyModelInfo) {
	colW := []int{8, 20, 12, 8, 8, 8, 8}
	headers := []string{
		i18n.T("cli.model.ls_header_id", nil),
		i18n.T("cli.model.ls_header_name", nil),
		i18n.T("cli.model.ls_header_type", nil),
		i18n.T("cli.model.ls_header_versions", nil),
		i18n.T("cli.model.ls_header_public", nil),
		i18n.T("cli.model.ls_header_used", nil),
		i18n.T("cli.model.ls_header_downloads", nil),
	}
	fmt.Fprintln(w, joinCells(headers, colW))
	for _, m := range models {
		allPublic := len(m.Versions) > 0
		for _, v := range m.Versions {
			if !v.Public {
				allPublic = false
				break
			}
		}
		publicLabel := i18n.T("cli.model.ls_header_public_no", nil)
		if allPublic && len(m.Versions) > 0 {
			publicLabel = i18n.T("cli.model.ls_header_public_yes", nil)
		}
		cells := []string{
			strconv.Itoa(int(m.Id)),
			m.Name,
			m.Type,
			strconv.Itoa(len(m.Versions)),
			publicLabel,
			strconv.Itoa(m.Counter.UsedCount),
			strconv.Itoa(m.Counter.DownloadedCount),
		}
		fmt.Fprintln(w, joinCells(cells, colW))
	}
}

// joinCells 将每列填充到 colW[i] 显示宽度后用空格拼接。
func joinCells(cells []string, colW []int) string {
	var b strings.Builder
	for i, c := range cells {
		dw := runewidth.StringWidth(c)
		b.WriteString(c)
		if i < len(cells)-1 {
			if dw < colW[i] {
				b.WriteString(strings.Repeat(" ", colW[i]-dw))
			}
			b.WriteByte(' ')
		}
	}
	return b.String()
}
