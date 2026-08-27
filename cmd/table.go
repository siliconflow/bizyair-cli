package cmd

import (
	"io"
	"os"
	"strconv"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

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

func tableColumnWidths(nCols, totalW int) []int {
	if nCols <= 0 {
		return nil
	}
	per := totalW / nCols
	rem := totalW % nCols
	widths := make([]int, nCols)
	for i := range widths {
		widths[i] = per
		if i == nCols-1 {
			widths[i] += rem
		}
	}
	return widths
}

const cellPadWidth = 2

func truncateCell(s string, maxW int) string {
	if runewidth.StringWidth(s) <= maxW {
		return s
	}
	const tail = "..."
	if maxW <= runewidth.StringWidth(tail) {
		return runewidth.Truncate(s, maxW, "")
	}
	return runewidth.Truncate(s, maxW, tail)
}

func newTable(w io.Writer, nCols int) *tablewriter.Table {
	colWs := tableColumnWidths(nCols, terminalWidth())
	widths := tw.NewMapper[int, int]()
	for i, cw := range colWs {
		widths = widths.Set(i, cw)
	}
	table := tablewriter.NewTable(w,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenColumns: tw.Off},
				Lines:      tw.Lines{ShowHeaderLine: tw.Off},
			},
		})),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithMaxWidth(terminalWidth()),
		tablewriter.WithColumnWidths(widths),
	)
	return table
}