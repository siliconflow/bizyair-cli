package actions

import (
	"context"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// Whoami 查询当前用户信息。
func Whoami(api lib.BizyAPI) WhoamiResult {
	return WhoamiContext(context.Background(), api)
}

// WhoamiContext 查询当前用户信息。
func WhoamiContext(ctx context.Context, api lib.BizyAPI) WhoamiResult {
	resp, err := api.UserInfoContext(ctx)
	if err != nil {
		return WhoamiResult{
			Error: lib.WithStep(i18n.T("step.whoami"), err),
		}
	}
	var user *lib.UserInfo
	if resp != nil {
		user = &resp.Data
	}
	return WhoamiResult{
		User: user,
	}
}

// Plan 查询套餐概览。
func Plan(api lib.BizyAPI) PlanResult {
	return PlanContext(context.Background(), api)
}

// PlanContext 查询套餐概览。
func PlanContext(ctx context.Context, api lib.BizyAPI) PlanResult {
	resp, err := api.GetPlanOverviewContext(ctx)
	if err != nil {
		return PlanResult{
			Error: lib.WithStep(i18n.T("step.plan"), err),
		}
	}
	var plan *lib.PlanOverviewResp
	if resp != nil {
		plan = &resp.Data
	}
	return PlanResult{
		Plan: plan,
	}
}

// Wallet 查询钱包余额。
func Wallet(api lib.BizyAPI) WalletResult {
	return WalletContext(context.Background(), api)
}

// WalletContext 查询钱包余额。
func WalletContext(ctx context.Context, api lib.BizyAPI) WalletResult {
	resp, err := api.GetWalletContext(ctx)
	if err != nil {
		return WalletResult{
			Error: lib.WithStep(i18n.T("step.wallet"), err),
		}
	}
	var wallet *lib.WalletInfo
	if resp != nil {
		wallet = &resp.Data.WalletInfo
	}
	return WalletResult{
		Wallet: wallet,
	}
}

// Credits 查询积分明细。
func Credits(api lib.BizyAPI) CreditsResult {
	return CreditsContext(context.Background(), api)
}

// CreditsContext 查询积分明细。
func CreditsContext(ctx context.Context, api lib.BizyAPI) CreditsResult {
	resp, err := api.GetCreditsContext(ctx, 1, 100, 0)
	if err != nil {
		return CreditsResult{
			Error: lib.WithStep(i18n.T("step.credits"), err),
		}
	}
	var credits []*lib.CreditItem
	var total int
	var giftAmount, rechargeAmount, totalAmount int64
	if resp != nil {
		credits = resp.Data.List
		total = resp.Data.Total
		giftAmount = resp.Data.GiftAmount
		rechargeAmount = resp.Data.RechargeAmount
		totalAmount = resp.Data.TotalAmount
	}
	return CreditsResult{
		Credits:        credits,
		Total:          total,
		GiftAmount:     giftAmount,
		RechargeAmount: rechargeAmount,
		TotalAmount:    totalAmount,
	}
}

// DayCost 查询今日消费记录。
func DayCost(api lib.BizyAPI) DayCostResult {
	return DayCostContext(context.Background(), api)
}

// DayCostContext 查询今日消费记录。
func DayCostContext(ctx context.Context, api lib.BizyAPI) DayCostResult {
	today := time.Now().UTC().Format("2006-01-02")
	resp, err := api.GetDayCostContext(ctx, today)
	if err != nil {
		return DayCostResult{
			Error: lib.WithStep(i18n.T("step.day_cost"), err),
		}
	}
	var records []*lib.CostByTimeResult
	if resp != nil {
		records = resp.Data.ByTimeResults
	}
	return DayCostResult{
		Records: records,
	}
}
