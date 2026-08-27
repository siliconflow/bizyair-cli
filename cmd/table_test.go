package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/siliconflow/bizyair-cli/lib"
)

func TestTableColumnWidths(t *testing.T) {
	cases := []struct {
		nCols  int
		totalW int
		want   []int
	}{
		{3, 100, []int{33, 33, 34}},
		{4, 40, []int{10, 10, 10, 10}},
		{1, 80, []int{80}},
		{0, 80, nil},
	}
	for _, c := range cases {
		got := tableColumnWidths(c.nCols, c.totalW)
		if len(got) != len(c.want) {
			t.Fatalf("tableColumnWidths(%d,%d) len=%d want %d", c.nCols, c.totalW, len(got), len(c.want))
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("tableColumnWidths(%d,%d)[%d]=%d want %d", c.nCols, c.totalW, i, got[i], c.want[i])
			}
		}
	}
}

func TestTruncateCell(t *testing.T) {
	cases := []struct {
		s    string
		maxW int
		want string
	}{
		{"abc", 10, "abc"},
		{"abcdefghij", 5, "ab..."},
		{"abcde", 5, "abcde"},
		{"中文测试", 5, "中..."},
		{"abcdef", 2, "ab"},
		{"", 10, ""},
	}
	for _, c := range cases {
		got := truncateCell(c.s, c.maxW)
		if got != c.want {
			t.Fatalf("truncateCell(%q,%d)=%q want %q", c.s, c.maxW, got, c.want)
		}
		if w := runewidth.StringWidth(got); w > c.maxW {
			t.Fatalf("truncateCell(%q,%d) width %d exceeds %d", c.s, c.maxW, w, c.maxW)
		}
	}
}

func TestPrintModelTableFillsWidth(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	models := []*lib.BizyModelInfo{{
		Id:   1,
		Name: "an-very-long-model-name-that-definitely-exceeds-column-width-and-needs-truncation",
		Type: "LoRA",
		Versions: []*lib.BizyModelVersion{{
			Public: true,
		}},
		Counter: lib.ModelCounter{UsedCount: 12, DownloadedCount: 3456},
	}}
	var buf bytes.Buffer
	printModelTable(&buf, models)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + data rows, got %d lines: %q", len(lines), buf.String())
	}
	for _, line := range lines {
		if line == "" {
			continue
		}
		if w := runewidth.StringWidth(line); w < 78 || w > 80 {
			t.Fatalf("line width %d out of [78,80]: %q", w, line)
		}
	}
	if !strings.Contains(buf.String(), "...") {
		t.Fatalf("expected ellipsis truncation marker, got:\n%s", buf.String())
	}
}

func TestPrintModelzooTableFillsWidth(t *testing.T) {
	t.Setenv("COLUMNS", "80")
	models := []lib.ModelzooModelFlat{{
		DisplayName:  "stable-diffusion-xl-base-1.0-huge-model-name",
		Endpoint:     "https://api.siliconflow.cn/v1/images/generations-very-long-endpoint-path",
		Manufacturer: "stabilityai",
		Category:     "Image Generation",
		ModelVersion: "SDXL base 1.0 very long version string",
	}}
	var buf bytes.Buffer
	printModelzooTable(&buf, models)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + data rows, got %d lines: %q", len(lines), buf.String())
	}
	for _, line := range lines {
		if line == "" {
			continue
		}
		if w := runewidth.StringWidth(line); w < 78 || w > 80 {
			t.Fatalf("line width %d out of [78,80]: %q", w, line)
		}
	}
	if !strings.Contains(buf.String(), "...") {
		t.Fatalf("expected ellipsis truncation marker, got:\n%s", buf.String())
	}
}