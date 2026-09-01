package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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

	if len(result.PriceTables) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_empty", nil))
		return nil
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.modelzoo.price_title", map[string]any{"Endpoint": endpoint}))
	for _, pt := range result.PriceTables {
		table := newTable(os.Stdout, len(priceColumns(pt))+2)
		table.Header(anyStrings(priceColumns(pt))...)
		for _, row := range priceRows(pt) {
			table.Append(row)
		}
		table.Render()
	}
	return nil
}

func priceColumns(pt lib.PriceTable) []string {
	cols := make([]string, 0, len(pt.Columns)+3)
	for _, col := range pt.Columns {
		title := i18n.APITranslate("param_label", col.FieldLabel)
		if title == "" {
			title = col.FieldName
		}
		if title == "" {
			title = "-"
		}
		cols = append(cols, title)
	}
	cols = append(cols, i18n.T("cli.modelzoo.price_unit_header", nil))
	cols = append(cols, i18n.T("cli.modelzoo.price_value_header", nil))
	if len(pt.Remarks) > 0 {
		cols = append(cols, i18n.T("cli.modelzoo.price_remark_header", nil))
	}
	return cols
}

func priceRows(pt lib.PriceTable) [][]string {
	n := len(pt.PricingValues)
	if len(pt.CellsV2) > n {
		n = len(pt.CellsV2)
	}
	if len(pt.Remarks) > n {
		n = len(pt.Remarks)
	}
	rows := make([][]string, 0, n)
	for r := 0; r < n; r++ {
		row := make([]string, 0, len(pt.Columns)+3)
		for c := 0; c < len(pt.Columns); c++ {
			cell := "-"
			if r < len(pt.CellsV2) && c < len(pt.CellsV2[r]) {
				cell = pt.CellsV2[r][c]
			}
			row = append(row, cell)
		}
		row = append(row, priceUnitDisplay(pt))
		val := "-"
		if r < len(pt.PricingValues) {
			val = formatPriceValue(pt.PricingValues[r])
		}
		row = append(row, val)
		if len(pt.Remarks) > 0 {
			remark := "-"
			if r < len(pt.Remarks) && strings.TrimSpace(pt.Remarks[r]) != "" {
				remark = i18n.APITranslate("remark", pt.Remarks[r])
			}
			row = append(row, remark)
		}
		rows = append(rows, row)
	}
	return rows
}

// priceUnitLabel 将 API 返回的计价单位（SECOND/CALL/IMAGE/MTOKEN 等）翻译为本地化文本。
func priceUnitLabel(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "SECOND", "SECONDS", "S", "PER_SECOND", "每秒", "秒":
		return i18n.T("cli.modelzoo.price_unit_per_second", nil)
	case "CALL", "CALLS", "PER_CALL", "每次", "次":
		return i18n.T("cli.modelzoo.price_unit_per_call", nil)
	case "IMAGE", "IMAGES", "IMG", "PER_IMAGE", "每张", "张":
		return i18n.T("cli.modelzoo.price_unit_per_image", nil)
	case "MTOKEN", "MTOK", "MT", "PER_MTOKEN", "PER_MT", "百万TOKEN":
		return i18n.T("cli.modelzoo.price_unit_per_mtoken", nil)
	default:
		return name
	}
}

// priceUnitDisplay 组合价格表类型与计价方式，如 “Input（按秒计费）”。
func priceUnitDisplay(pt lib.PriceTable) string {
	unit := priceUnitLabel(pt.UnitName)
	if pt.PricingName != "" {
		if unit != "" {
			return pt.PricingName + "（" + unit + "）"
		}
		return pt.PricingName
	}
	if unit == "" {
		return "-"
	}
	return unit
}

// formatPriceValue 将价格数值（积分，千分之一美元）格式化为美元金额显示，如 “$1.234”。
// 支持 “数值*倍数” 形式（如 “16*4”），仅转换数值部分。
func formatPriceValue(val string) string {
	trimmed := strings.TrimSpace(val)
	if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return usdPrice(v)
	}
	parts := strings.SplitN(val, "*", 2)
	if len(parts) == 2 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			return usdPrice(v) + " *" + parts[1]
		}
	}
	return val
}

// usdPrice 将积分值（千分之一美元）格式化为美元金额字符串。
func usdPrice(credits float64) string {
	return fmt.Sprintf("%s%g", i18n.T("common.currency_usd", nil), credits/1000.0)
}

// anyStrings converts a []string to []any for the variadic Header API.
func anyStrings(in []string) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}
