package domain

// UserInfo 当前登录用户的基本信息。
type UserInfo struct {
	Id               string `json:"id"`
	NickName         string `json:"nick_name"`
	Avatar           string `json:"avatar"`
	Email            string `json:"email"`
	Status           string `json:"status"`
	Description      string `json:"description"`
	Level            int    `json:"level"`
	LevelDisplayName string `json:"level_display_name"`
	Language         string `json:"language"`
	OnlineTester     bool   `json:"online_tester"`
	APIKey           string `json:"api_key"`
}

// PlanInfo 套餐信息。
type PlanInfo struct {
	EffectiveDisplayName     string `json:"effective_display_name,omitempty"`
	AutoDisplayName          string `json:"auto_display_name,omitempty"`
	EffectiveRechargeCredits int64  `json:"effective_recharge_credits,omitempty"`
	NextDowngradeCheckAt     string `json:"next_downgrade_check_at,omitempty"`
}

// PlanNextTier 下一档套餐信息。
type PlanNextTier struct {
	DisplayName          string `json:"display_name,omitempty"`
	NeedRechargeUsdCents int64  `json:"need_recharge_usd_cents,omitempty"`
}

// PlanWalletInfo 套餐内的钱包余额信息。
type PlanWalletInfo struct {
	GiftBalance           int64 `json:"gift_balance,omitempty"`
	RechargeBalance       int64 `json:"recharge_balance,omitempty"`
	CreditsExpireDays     int   `json:"credits_expire_days,omitempty"`
	GiftCreditsExpireDays int   `json:"gift_credits_expire_days,omitempty"`
}

// PlanRollingWindow 滚动窗口内的额度变化信息。
type PlanRollingWindow struct {
	EffectiveRechargeCredits30d int64 `json:"effective_recharge_credits_30d,omitempty"`
	ExpiringRechargeCredits1d   int64 `json:"expiring_recharge_credits_1d,omitempty"`
	ExpiringRechargeCredits7d   int64 `json:"expiring_recharge_credits_7d,omitempty"`
}

// PlanOverviewResp 套餐概览响应。
type PlanOverviewResp struct {
	Plan          *PlanInfo          `json:"plan,omitempty"`
	NextTier      *PlanNextTier      `json:"next_tier,omitempty"`
	Wallet        *PlanWalletInfo    `json:"wallet,omitempty"`
	RollingWindow *PlanRollingWindow `json:"rolling_window,omitempty"`
}

// WalletInfo 钱包余额信息。
type WalletInfo struct {
	GiftBalance           int64 `json:"gift_balance,omitempty"`
	RechargeBalance       int64 `json:"recharge_balance,omitempty"`
	TotalAvailableCoupons int   `json:"total_available_coupons,omitempty"`
	CreditsTodayExpired   bool  `json:"credits_today_expired,omitempty"`
}

// WalletResp 是 /v1/wallet 的响应：钱包字段平铺在 data 层
// （与 PlanOverviewResp 的 wallet 嵌套不同），内嵌 WalletInfo 直接吸收。
type WalletResp struct {
	WalletInfo
}

// CreditItem 单笔积分明细。
type CreditItem struct {
	TotalAmount    int64  `json:"total_amount,omitempty"`
	GiftAmount     int64  `json:"gift_amount,omitempty"`
	RechargeAmount int64  `json:"recharge_amount,omitempty"`
	ExpiredAt      string `json:"expired_at,omitempty"`
}

// CreditsListResp 积分明细列表响应。
type CreditsListResp struct {
	List           []*CreditItem `json:"list,omitempty"`
	Total          int           `json:"total,omitempty"`
	GiftAmount     int64         `json:"gift_amount,omitempty"`
	RechargeAmount int64         `json:"recharge_amount,omitempty"`
	TotalAmount    int64         `json:"total_amount,omitempty"`
}

// CreditsReq 积分明细查询参数。
type CreditsReq struct {
	Current    int `form:"current" query:"current"`
	PageSize   int `form:"page_size" query:"page_size"`
	ExpireDays int `form:"expire_days" query:"expire_days"`
}

// CostByTimeResult 按时间段的消费记录。
type CostByTimeResult struct {
	Time           string `json:"time,omitempty"`
	CreditsAmount  int64  `json:"credits_amount,omitempty"`
	PromptQuantity int64  `json:"prompt_quantity,omitempty"`
}

// DayCostResp 每日消费响应。
type DayCostResp struct {
	ByTimeResults []*CostByTimeResult `json:"by_time_results,omitempty"`
}
