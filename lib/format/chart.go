package format

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// HourCost 单个小时内的消费汇总。
type HourCost struct {
	Hour    int
	Credits int64
	Prompts int64
}

// DayCostChart 绘制每小时消费柱状图：每行一个小时的消费金额与请求数。
func DayCostChart(records []HourCost, barWidth int) string {
	if len(records) == 0 {
		return ""
	}

	var maxCredits int64
	for _, r := range records {
		if r.Credits > maxCredits {
			maxCredits = r.Credits
		}
	}

	var totalCredits int64
	var totalPrompts int64
	var b strings.Builder

	sorted := make([]HourCost, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Hour < sorted[j].Hour
	})

	for _, r := range sorted {
		totalCredits += r.Credits
		totalPrompts += r.Prompts

		label := fmt.Sprintf("%02d:00", r.Hour)
		costStr := fmt.Sprintf("%-8s", formatCredits(r.Credits))

		barLen := 0
		if maxCredits > 0 {
			barLen = int(float64(r.Credits) / float64(maxCredits) * float64(barWidth))
		}
		bar := strings.Repeat("█", barLen)

		fmt.Fprint(&b, i18n.T("cli.chart.row_format", map[string]any{
			"Time": label, "Cost": costStr, "Bar": bar, "Requests": r.Prompts,
		})+"\n")
	}

	fmt.Fprint(&b, i18n.T("cli.chart.total_format", map[string]any{
		"Cost": formatCredits(totalCredits), "Requests": totalPrompts,
	})+"\n")

	return b.String()
}

// ExtractHourCosts 将时间串与消费/请求数序列按小时聚合为 HourCost 列表。
func ExtractHourCosts(isoTimes []string, creditsAmounts []int64, promptQuantities []int64) []HourCost {
	if len(isoTimes) == 0 {
		return nil
	}

	groups := make(map[int]*HourCost)

	for i, t := range isoTimes {
		hour := parseHour(t)
		if _, ok := groups[hour]; !ok {
			groups[hour] = &HourCost{Hour: hour}
		}
		if i < len(creditsAmounts) {
			groups[hour].Credits += creditsAmounts[i]
		}
		if i < len(promptQuantities) {
			groups[hour].Prompts += promptQuantities[i]
		}
	}

	result := make([]HourCost, 0, len(groups))
	for _, v := range groups {
		result = append(result, *v)
	}
	return result
}

// formatCredits 将积分（千分之一美元）格式化为美元金额显示。
func formatCredits(credits int64) string {
	dollars := float64(credits) / 1000.0
	sym := i18n.T("common.currency_usd", nil)
	return fmt.Sprintf("%s%g", sym, dollars)
}

// parseHour 从时间串中解析小时：带时区按本地时区取小时，无时区按本地时区解析。
func parseHour(isoTime string) int {
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
	} {
		if t, err := time.Parse(layout, isoTime); err == nil {
			return t.Local().Hour()
		}
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, isoTime, time.Local); err == nil {
			return t.Hour()
		}
	}
	return 0
}
