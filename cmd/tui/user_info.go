// 本文件实现用户信息视图的渲染：
// 用户信息/套餐/积分/每日消费的展示，含等级颜色映射、余额卡片与消费柱状图。
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
)

// levelKeyAliases 将等级显示名的常见变体（英文标识、中文显示名）映射为规范等级标识。
var levelKeyAliases = map[string]string{
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

// creditsToUSD 将积分（千分之一美元）格式化为美元金额显示。
func creditsToUSD(credits int64) string {
	return fmt.Sprintf("%s%.3f", i18n.T("common.currency_usd", nil), float64(credits)/1000.0)
}

// renderWhoami 渲染当前用户信息为键值表格（ID/昵称/邮箱/状态）。
func (m mainModel) renderWhoami(user *lib.UserInfo) string {
	if user == nil {
		return i18n.T("error.auth.not_logged_in_simple", nil)
	}
	labels := []string{
		i18n.T("cli.whoami.id_label", nil),
		i18n.T("cli.whoami.name_label", nil),
		i18n.T("cli.whoami.email_label", nil),
		i18n.T("cli.whoami.status_label", nil),
	}
	values := []string{
		dash(user.Id),
		dash(user.NickName),
		dash(user.Email),
		formatUserStatusTUI(user.Status),
	}
	whoamiMaxW := m.width - 16
	if whoamiMaxW < 60 {
		whoamiMaxW = 60
	}
	return renderKeyValueTable(labels, values, whoamiMaxW)
}

// formatUserStatusTUI 将用户状态翻译并着色（normal 绿/banned 红/pending 黄）。
func formatUserStatusTUI(status string) string {
	if status == "" {
		return "-"
	}
	translated := i18n.APITranslate("user_status", status)
	switch strings.ToLower(status) {
	case "normal":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(translated)
	case "banned", "frozen":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(translated)
	case "pending", "inactive":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render(translated)
	default:
		return translated
	}
}

// normalizeLevelKey 将 API 返回的等级名（如 "Basic"、"基础"）规范化为等级标识
// （basic/iron/bronze/silver/gold/platinum）；无法识别时返回空字符串。
func normalizeLevelKey(name string) string {
	return levelKeyAliases[strings.ToLower(strings.TrimSpace(name))]
}

// levelColor 返回各等级的显示颜色，未知等级返回空颜色。
func levelColor(name string) lipgloss.Color {
	switch normalizeLevelKey(name) {
	case "basic":
		return lipgloss.Color("#94A3B8")
	case "iron":
		return lipgloss.Color("#6B7280")
	case "bronze":
		return lipgloss.Color("#D97706")
	case "silver":
		return lipgloss.Color("#E5E7EB")
	case "gold":
		return lipgloss.Color("#FACC15")
	case "platinum":
		return lipgloss.Color("#CFFAFE")
	default:
		return lipgloss.Color("")
	}
}

// levelDisplayName 返回等级的本地化显示名称，无法识别时原样返回。
func levelDisplayName(name string) string {
	key := normalizeLevelKey(name)
	if key == "" {
		return name
	}
	translated := i18n.T("cli.plan.level_name."+key, nil)
	if strings.HasPrefix(translated, "[missing translation:") {
		return name
	}
	return translated
}

// renderPlan 渲染套餐信息：当前套餐/降级检查时间/下一档套餐/充值金额/钱包余额。
func (m mainModel) renderPlan(plan *lib.PlanOverviewResp) string {
	if plan == nil {
		return i18n.T("error.auth.not_logged_in_simple", nil)
	}
	var labels []string
	var values []string
	if plan.Plan != nil {
		levelName := plan.Plan.EffectiveDisplayName
		displayName := levelDisplayName(levelName)
		if c := levelColor(plan.Plan.EffectiveDisplayName); c != "" {
			displayName = lipgloss.NewStyle().Foreground(c).Render(displayName)
		}
		labels = append(labels, i18n.T("cli.plan.effective_plan_label", nil))
		values = append(values, displayName)
		if plan.Plan.NextDowngradeCheckAt != "" {
			labels = append(labels, i18n.T("cli.plan.next_downgrade_label", nil))
			values = append(values, plan.Plan.NextDowngradeCheckAt)
		}
	}
	if plan.NextTier != nil {
		tierName := levelDisplayName(plan.NextTier.DisplayName)
		if c := levelColor(plan.NextTier.DisplayName); c != "" {
			tierName = lipgloss.NewStyle().Foreground(c).Render(tierName)
		}
		labels = append(labels, i18n.T("cli.plan.next_tier_label", nil))
		values = append(values, tierName)
		if plan.NextTier.NeedRechargeUsdCents > 0 {
			labels = append(labels, i18n.T("cli.plan.need_recharge_label", nil))
			values = append(values, fmt.Sprintf("%s%.3f", i18n.T("common.currency_usd", nil), float64(plan.NextTier.NeedRechargeUsdCents)/100.0))
		}
	}
	if plan.Wallet != nil {
		labels = append(labels, i18n.T("cli.wallet.recharge_balance_label", nil))
		values = append(values, creditsToUSD(plan.Wallet.RechargeBalance))
		labels = append(labels, i18n.T("cli.wallet.gift_balance_label", nil))
		values = append(values, creditsToUSD(plan.Wallet.GiftBalance))
	}
	if len(labels) == 0 {
		return ""
	}
	maxW := m.width - 16
	if maxW < 60 {
		maxW = 60
	}
	return renderKeyValueTable(labels, values, maxW)
}

// earliestExpiry 返回充值/赠送积分中最早的有效期时间串，用于提示到期日。
func earliestExpiry(credits []*lib.CreditItem, isGift bool) string {
	var earliest string
	for _, c := range credits {
		if c.ExpiredAt == "" {
			continue
		}
		matched := (isGift && c.GiftAmount > 0) || (!isGift && c.RechargeAmount > 0)
		if !matched {
			continue
		}
		if earliest == "" || c.ExpiredAt < earliest {
			earliest = c.ExpiredAt
		}
	}
	return earliest
}

// formatExpiry 将到期时间格式化为"日期（剩余天数）"，已过期标注过期。
func formatExpiry(expiredAt string) string {
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

// renderCredits 渲染积分视图卡片：
// 顶部总余额，下方并排（或纵向堆叠）展示充值/赠送余额及各自最早到期日。
func (m mainModel) renderCredits(credits []*lib.CreditItem, total int, giftAmt, rechargeAmt, totalAmt int64) string {
	balance := totalAmt
	if balance == 0 {
		balance = rechargeAmt + giftAmt
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7"))
	amountStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FACC15"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	giftExp := earliestExpiry(credits, true)
	rechargeExp := earliestExpiry(credits, false)

	subCardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2)

	innerW, innerH := m.innerSize()
	// panel 内容区宽度 = innerW-2(border)-4(padding) = innerW-6；
	// 卡片总宽 = outerW+2(border)，须 ≤ innerW-6，即 outerW ≤ innerW-8。
	outerW := innerW - 10
	if outerW < 18 {
		outerW = 18
	}
	panelH := innerH - 13
	if panelH < 8 {
		panelH = 8
	}
	if panelH > 38 {
		panelH = 38
	}
	subW := (outerW - 10) / 2
	sideBySide := subW >= 16
	if !sideBySide {
		subW = outerW - 8
	}
	subCardStyle = subCardStyle.Width(subW)

	leftContent := titleStyle.Render(i18n.T("cli.credits.recharge_balance", nil)) + "\n" +
		amountStyle.Render(creditsToUSD(rechargeAmt)) + "\n"
	if rechargeExp != "" {
		leftContent += dimStyle.Render(formatExpiry(rechargeExp)) + "\n"
	}
	leftCard := subCardStyle.Render(leftContent)

	rightContent := titleStyle.Render(i18n.T("cli.credits.gift_balance", nil)) + "\n" +
		amountStyle.Render(creditsToUSD(giftAmt)) + "\n"
	if giftExp != "" {
		rightContent += dimStyle.Render(formatExpiry(giftExp)) + "\n"
	}
	rightCard := subCardStyle.Render(rightContent)

	// 并排布局时让两侧卡片等高
	if sideBySide {
		leftH := lipgloss.Height(leftCard)
		rightH := lipgloss.Height(rightCard)
		maxSubH := leftH
		if rightH > maxSubH {
			maxSubH = rightH
		}
		subCardStyle = subCardStyle.Height(maxSubH)
		leftCard = subCardStyle.Render(leftContent)
		rightCard = subCardStyle.Render(rightContent)
	}

	var subCards string
	if sideBySide {
		subCards = lipgloss.JoinHorizontal(lipgloss.Top, leftCard, rightCard)
	} else {
		subCards = leftCard + "\n" + rightCard
	}

	inner := titleStyle.Render(i18n.T("cli.credits.current_balance", nil)) + "\n" +
		amountStyle.Render(creditsToUSD(balance)) + "\n\n" +
		subCards + "\n"

	outerCard := lipgloss.NewStyle().
		Padding(1, 2).
		Width(outerW).
		Height(panelH).
		Render(inner)

	return outerCard
}

// renderDayCost 渲染每日消费视图：
// 汇总每小时的消费数据并绘制 ASCII 柱状图，底部展示总积分与总请求数。
func (m mainModel) renderDayCost(records []*lib.CostByTimeResult) string {
	if len(records) == 0 {
		return i18n.T("cli.day_cost.no_records", nil)
	}
	var b strings.Builder
	var times []string
	var creditAmounts []int64
	var promptQuantities []int64
	var totalCredits int64
	var totalPrompts int64
	for _, r := range records {
		times = append(times, r.Time)
		creditAmounts = append(creditAmounts, r.CreditsAmount)
		promptQuantities = append(promptQuantities, r.PromptQuantity)
		totalCredits += r.CreditsAmount
		totalPrompts += r.PromptQuantity
	}
	hours := format.ExtractHourCosts(times, creditAmounts, promptQuantities)

	innerW, _ := m.innerSize()
	barW := innerW - 30
	if barW < 10 {
		barW = 10
	}
	if barW > 40 {
		barW = 40
	}

	b.WriteString(m.hintStyle.Render(i18n.T("cli.chart.intro", nil)))
	b.WriteString("\n\n")
	b.WriteString(format.DayCostChart(hours, barW))
	b.WriteString("\n")
	b.WriteString(i18n.T("cli.day_cost.total_credits_label", nil) + ": " + creditsToUSD(totalCredits) + "\n")
	b.WriteString(i18n.T("cli.day_cost.total_prompts_label", nil) + ": " + fmt.Sprintf("%d", totalPrompts) + "\n")
	return b.String()
}
