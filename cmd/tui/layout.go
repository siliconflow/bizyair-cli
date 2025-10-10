package tui

import (
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *mainModel) computeInnerSizeFor(totalW, totalH int) (int, int) {
	iw := totalW - 2 - m.framePadX*2
	ih := totalH - 2 - m.framePadY*2
	if iw < 1 {
		iw = 1
	}
	if ih < 1 {
		ih = 1
	}
	return iw, ih
}

func (m *mainModel) innerSize() (int, int) { return m.computeInnerSizeFor(m.width, m.height) }

func clipToWidth(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

func ensureTrailingSep(p string) string {
	sep := string(filepath.Separator)
	if strings.HasSuffix(p, sep) {
		return p
	}
	return p + sep
}

// 全屏渐变外框
func (m *mainModel) renderFrame(inner string) string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return inner
	}
	iw, ih := m.innerSize()
	padX, padY := m.framePadX, m.framePadY

	lines := strings.Split(inner, "\n")
	contentLines := make([]string, ih)
	for i := 0; i < ih; i++ {
		if i < len(lines) {
			contentLines[i] = clipToWidth(lines[i], iw)
		} else {
			contentLines[i] = clipToWidth("", iw)
		}
	}

	sr, sg, sb := hexToRGB("#8B5CF6")
	er, eg, eb := hexToRGB("#EC4899")

	var b strings.Builder

	if h >= 1 {
		tColor := rgbToHex(sr, sg, sb)
		tStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tColor))
		if w >= 2 {
			b.WriteString(tStyle.Render("╭" + strings.Repeat("─", w-2) + "╮"))
		} else {
			b.WriteString(tStyle.Render("╭"))
		}
		b.WriteString("\n")
	}

	innerStartY := padY
	innerEndY := padY + ih
	for y := 1; y <= h-2; y++ {
		t := 0.0
		if h > 1 {
			t = float64(y) / float64(h-1)
		}
		r := lerpInt(sr, er, t)
		g := lerpInt(sg, eg, t)
		bl := lerpInt(sb, eb, t)
		c := rgbToHex(r, g, bl)
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(c))

		middle := ""
		if w >= 2 {
			if y-1 < innerStartY || y-1 >= innerEndY {
				middle = strings.Repeat(" ", w-2)
			} else {
				row := y - 1 - innerStartY
				if row >= 0 && row < len(contentLines) {
					middle = strings.Repeat(" ", padX) + clipToWidth(contentLines[row], iw) + strings.Repeat(" ", padX)
				} else {
					middle = strings.Repeat(" ", w-2)
				}
			}
		}

		if w >= 2 {
			b.WriteString(s.Render("│") + middle + s.Render("│"))
		} else {
			b.WriteString(s.Render("│"))
		}
		b.WriteString("\n")
	}

	if h >= 2 {
		bColor := rgbToHex(er, eg, eb)
		bStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(bColor))
		if w >= 2 {
			b.WriteString(bStyle.Render("╰" + strings.Repeat("─", w-2) + "╯"))
		} else {
			b.WriteString(bStyle.Render("╰"))
		}
	}
	return b.String()
}

func rgbToHex(r, g, b int) string { return "#" + toHex(r) + toHex(g) + toHex(b) }

func toHex(v int) string {
	h := strconv.FormatInt(int64(v), 16)
	if len(h) == 1 {
		h = "0" + h
	}
	return strings.ToUpper(h)
}

func hexToRGB(hex string) (int, int, int) {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return 255, 255, 255
	}
	r, _ := strconv.ParseInt(s[0:2], 16, 0)
	g, _ := strconv.ParseInt(s[2:4], 16, 0)
	b, _ := strconv.ParseInt(s[4:6], 16, 0)
	return int(r), int(g), int(b)
}

func lerpInt(a, b int, t float64) int {
	x := float64(a) + (float64(b)-float64(a))*t
	if x < 0 {
		x = 0
	}
	if x > 255 {
		x = 255
	}
	return int(math.Round(x))
}
