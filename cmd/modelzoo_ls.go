package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/charmbracelet/x/term"
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
	headers := []string{
		i18n.T("cli.modelzoo.ls_header_name", nil),
		i18n.T("cli.modelzoo.ls_header_endpoint", nil),
		i18n.T("cli.modelzoo.ls_header_manufacturer", nil),
		i18n.T("cli.modelzoo.ls_header_category", nil),
		i18n.T("cli.modelzoo.ls_header_version", nil),
	}
	colW := equalColWidths(terminalWidth(), len(headers))
	fmt.Fprintln(w, joinCells(headers, colW))
	for _, m := range models {
		cells := []string{
			dashIfEmpty(m.DisplayName),
			m.Endpoint,
			dashIfEmpty(i18n.APITranslate("manufacturer", m.Manufacturer)),
			dashIfEmpty(m.Category),
			dashIfEmpty(i18n.APITranslate("version", m.ModelVersion)),
		}
		fmt.Fprintln(w, joinCells(cells, colW))
	}
}

func terminalWidth() int {
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	if c := os.Getenv("COLUMNS"); c != "" {
		if w, err := strconv.Atoi(c); err == nil && w > 0 {
			return w
		}
	}
	return 120
}

func equalColWidths(available, n int) []int {
	if n <= 0 {
		return nil
	}
	gapTotal := n - 1
	per := (available - gapTotal) / n
	rem := (available - gapTotal) % n
	const minW = 8
	if per < minW {
		per, rem = minW, 0
	}
	colW := make([]int, n)
	for i := range colW {
		colW[i] = per
		if i < rem {
			colW[i]++
		}
	}
	return colW
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
