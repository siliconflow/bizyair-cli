package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
)

func formatUserStatusTUI(status string) string {
	text := format.UserStatusText(status)
	hex := format.UserStatusHexColor(status)
	if hex != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(text)
	}
	return text
}

func levelColor(name string) lipgloss.Color {
	return lipgloss.Color(format.LevelHexColor(name))
}

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

func (m mainModel) renderPlan(plan *lib.PlanOverviewResp) string {
	if plan == nil {
		return i18n.T("error.auth.not_logged_in_simple", nil)
	}
	var labels []string
	var values []string
	if plan.Plan != nil {
		levelName := plan.Plan.EffectiveDisplayName
		displayName := format.LevelDisplayName(levelName)
		if c := levelColor(plan.Plan.EffectiveDisplayName); c != "" {
			displayName = lipgloss.NewStyle().Foreground(c).Render(displayName)
		}
		labels = append(labels, i18n.T("cli.plan.effective_plan_label", nil))
		values = append(values, displayName)
		if plan.Plan.NextDowngradeCheckAt != "" {
			labels = append(labels, i18n.T("cli.plan.next_downgrade_label", nil))
			values = append(values, format.FormatTimestamp(plan.Plan.NextDowngradeCheckAt))
		}
	}
	if plan.NextTier != nil {
		tierName := format.LevelDisplayName(plan.NextTier.DisplayName)
		if c := levelColor(plan.NextTier.DisplayName); c != "" {
			tierName = lipgloss.NewStyle().Foreground(c).Render(tierName)
		}
		labels = append(labels, i18n.T("cli.plan.next_tier_label", nil))
		values = append(values, tierName)
		if plan.NextTier.NeedRechargeUsdCents > 0 {
			labels = append(labels, i18n.T("cli.plan.need_recharge_label", nil))
			values = append(values, format.FormatUsdCents(plan.NextTier.NeedRechargeUsdCents))
		}
	}
	if plan.Wallet != nil {
		labels = append(labels, i18n.T("cli.wallet.recharge_balance_label", nil))
		values = append(values, format.FormatCredits(plan.Wallet.RechargeBalance))
		labels = append(labels, i18n.T("cli.wallet.gift_balance_label", nil))
		values = append(values, format.FormatCredits(plan.Wallet.GiftBalance))
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

func (m mainModel) renderCredits(credits []*lib.CreditItem, total int, giftAmt, rechargeAmt, totalAmt int64) string {
	balance := totalAmt

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7"))
	amountStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FACC15"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	giftExp := format.EarliestExpiry(credits, true)
	rechargeExp := format.EarliestExpiry(credits, false)

	subCardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2)

	innerW, innerH := m.innerSize()
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
		amountStyle.Render(format.FormatCredits(rechargeAmt)) + "\n"
	if rechargeExp != "" {
		leftContent += dimStyle.Render(format.FormatExpiry(rechargeExp)) + "\n"
	}
	leftCard := subCardStyle.Render(leftContent)

	rightContent := titleStyle.Render(i18n.T("cli.credits.gift_balance", nil)) + "\n" +
		amountStyle.Render(format.FormatCredits(giftAmt)) + "\n"
	if giftExp != "" {
		rightContent += dimStyle.Render(format.FormatExpiry(giftExp)) + "\n"
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
		amountStyle.Render(format.FormatCredits(balance)) + "\n\n" +
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
	hours, totalCredits, totalPrompts := format.SummarizeDayCost(records)

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
	b.WriteString(i18n.T("cli.day_cost.total_credits_label", nil) + ": " + format.FormatCredits(totalCredits) + "\n")
	b.WriteString(i18n.T("cli.day_cost.total_prompts_label", nil) + ": " + fmt.Sprintf("%d", totalPrompts) + "\n")
	return b.String()
}
