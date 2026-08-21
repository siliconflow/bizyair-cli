// 本文件封装用户信息相关的 Bubble Tea 异步命令：
// 当前用户信息/套餐/积分/每日消费查询。
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// fetchWhoami 异步查询当前用户信息，结果以 whoamiDoneMsg 消息返回。
func fetchWhoami(api lib.BizyAPI) tea.Cmd {
	return func() tea.Msg {
		result := actions.WhoamiContext(context.Background(), api)
		return whoamiDoneMsg{user: result.User, err: result.Error}
	}
}

// fetchPlan 异步查询当前套餐信息，结果以 planDoneMsg 消息返回。
func fetchPlan(api lib.BizyAPI) tea.Cmd {
	return func() tea.Msg {
		result := actions.PlanContext(context.Background(), api)
		return planDoneMsg{plan: result.Plan, err: result.Error}
	}
}

// fetchCredits 异步查询积分明细，结果以 creditsDoneMsg 消息返回。
func fetchCredits(api lib.BizyAPI) tea.Cmd {
	return func() tea.Msg {
		result := actions.CreditsContext(context.Background(), api)
		return creditsDoneMsg{
			credits:     result.Credits,
			total:       result.Total,
			giftAmt:     result.GiftAmount,
			rechargeAmt: result.RechargeAmount,
			totalAmt:    result.TotalAmount,
			err:         result.Error,
		}
	}
}

// fetchDayCost 异步查询每日消费记录，结果以 dayCostDoneMsg 消息返回。
func fetchDayCost(api lib.BizyAPI) tea.Cmd {
	return func() tea.Msg {
		result := actions.DayCostContext(context.Background(), api)
		return dayCostDoneMsg{records: result.Records, err: result.Error}
	}
}
