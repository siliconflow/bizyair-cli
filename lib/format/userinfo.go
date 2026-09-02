package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/domain"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

var LevelKeyAliases = map[string]string{
	"basic":    "basic",
	"基础":       "basic",
	"iron":     "iron",
	"黑铁":       "iron",
	"bronze":   "bronze",
	"青铜":       "bronze",
	"silver":   "silver",
	"白银":       "silver",
	"gold":     "gold",
	"黄金":       "gold",
	"platinum": "platinum",
	"铂金":       "platinum",
}

var LevelHexColors = map[string]string{
	"basic":    "#94A3B8",
	"iron":     "#6B7280",
	"bronze":   "#D97706",
	"silver":   "#E5E7EB",
	"gold":     "#FACC15",
	"platinum": "#CFFAFE",
}

var UserStatusHexColors = map[string]string{
	"normal":   "#22C55E",
	"banned":   "#EF4444",
	"frozen":   "#EF4444",
	"pending":  "#F59E0B",
	"inactive": "#F59E0B",
}

func NormalizeLevelKey(name string) string {
	return LevelKeyAliases[strings.ToLower(strings.TrimSpace(name))]
}

func LevelDisplayName(name string) string {
	key := NormalizeLevelKey(name)
	if key == "" {
		return name
	}
	translated := i18n.T("cli.plan.level_name."+key, nil)
	if strings.HasPrefix(translated, "[missing translation:") {
		return name
	}
	return translated
}

func LevelHexColor(name string) string {
	return LevelHexColors[NormalizeLevelKey(name)]
}

func UserStatusText(status string) string {
	if status == "" {
		return "-"
	}
	return i18n.APITranslate("user_status", status)
}

func UserStatusHexColor(status string) string {
	return UserStatusHexColors[strings.ToLower(strings.TrimSpace(status))]
}

func FormatCredits(credits int64) string {
	sym := i18n.T("common.currency_usd", nil)
	return fmt.Sprintf("%s%.3f", sym, float64(credits)/1000.0)
}

func FormatUsdCents(cents int64) string {
	sym := i18n.T("common.currency_usd", nil)
	return fmt.Sprintf("%s%.2f", sym, float64(cents)/100.0)
}

func EarliestExpiry(credits []*domain.CreditItem, isGift bool) string {
	item := EarliestExpiryItem(credits, isGift)
	if item == nil {
		return ""
	}
	return FormatExpiry(item.ExpiredAt)
}

// EarliestExpiryItem 返回最早到期的积分条目（含金额），用于展示过期金额。
func EarliestExpiryItem(credits []*domain.CreditItem, isGift bool) *domain.CreditItem {
	var earliest *domain.CreditItem
	for _, c := range credits {
		if c.ExpiredAt == "" {
			continue
		}
		matched := (isGift && c.GiftAmount > 0) || (!isGift && c.RechargeAmount > 0)
		if !matched {
			continue
		}
		if earliest == nil || c.ExpiredAt < earliest.ExpiredAt {
			earliest = c
		}
	}
	return earliest
}

func FormatExpiry(expiredAt string) string {
	if expiredAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, expiredAt)
	if err != nil {
		t, err = time.Parse("2006-01-02", expiredAt)
		if err != nil {
			return ""
		}
	}
	dateStr := t.Format("2006-01-02")
	days := int(time.Until(t).Hours() / 24)
	if days <= 0 {
		return i18n.T("cli.credits.expired_with_date", map[string]any{"Date": dateStr})
	}
	return i18n.T("cli.credits.expire_with_date", map[string]any{"Date": dateStr, "Days": days})
}

// FormatExpiryLine 格式化余额过期行：最近一笔费用过期时间与过期金额。
func FormatExpiryLine(credits []*domain.CreditItem, isGift bool, balance string) string {
	item := EarliestExpiryItem(credits, isGift)
	if item == nil || item.ExpiredAt == "" {
		return balance
	}
	t, err := time.Parse(time.RFC3339, item.ExpiredAt)
	if err != nil {
		t, err = time.Parse("2006-01-02", item.ExpiredAt)
	}
	if err != nil {
		return balance
	}
	dateStr := t.Format("2006-01-02")
	days := int(time.Until(t).Hours() / 24)
	expireAmount := item.RechargeAmount
	if isGift {
		expireAmount = item.GiftAmount
	}
	if days <= 0 {
		return i18n.T("cli.credits.expired_line", map[string]any{
			"Balance": balance,
			"Date":    dateStr,
			"Amount":  FormatCredits(expireAmount),
		})
	}
	return i18n.T("cli.credits.expire_line", map[string]any{
		"Balance": balance,
		"Date":    dateStr,
		"Days":    days,
		"Amount":  FormatCredits(expireAmount),
	})
}

func FormatTimestamp(ts string) string {
	if ts == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t, err = time.Parse("2006-01-02", ts)
		if err != nil {
			return ts
		}
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func SummarizeDayCost(records []*domain.CostByTimeResult) (hours []HourCost, totalCredits, totalPrompts int64) {
	if len(records) == 0 {
		return nil, 0, 0
	}
	times := make([]string, 0, len(records))
	creditAmounts := make([]int64, 0, len(records))
	promptQuantities := make([]int64, 0, len(records))
	for _, r := range records {
		times = append(times, r.Time)
		creditAmounts = append(creditAmounts, r.CreditsAmount)
		promptQuantities = append(promptQuantities, r.PromptQuantity)
		totalCredits += r.CreditsAmount
		totalPrompts += r.PromptQuantity
	}
	hours = ExtractHourCosts(times, creditAmounts, promptQuantities)
	return hours, totalCredits, totalPrompts
}

func TodayDate() string {
	return time.Now().Format("2006-01-02")
}
