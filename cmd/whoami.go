// 本文件实现用户信息、套餐、钱包、积分与每日消费相关的 CLI 命令，
// 以及金额/时间/状态等展示格式化辅助函数。
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// formatCredits 将积分（千分之一美元）格式化为美元金额显示。
func formatCredits(credits int64) string {
	return formatUsdFromCredits(float64(credits) / 1000.0)
}

// formatUsdFromCredits 将美元金额格式化为带货币符号的字符串：
// 整数值不带小数位，否则保留三位小数。
func formatUsdFromCredits(usd float64) string {
	sym := i18n.T("common.currency_usd", nil)
	if usd == float64(int64(usd)) {
		return fmt.Sprintf("%s%d", sym, int64(usd))
	}
	return fmt.Sprintf("%s%.3f", sym, usd)
}

// formatUsdCents 将美分金额格式化为美元字符串，保留两位小数。
func formatUsdCents(cents int64) string {
	sym := i18n.T("common.currency_usd", nil)
	dollars := float64(cents) / 100.0
	if dollars == float64(int64(dollars)) {
		return fmt.Sprintf("%s%d", sym, int64(dollars))
	}
	return fmt.Sprintf("%s%.2f", sym, dollars)
}

// formatUserStatusCLI 将用户状态翻译为本地化文案，并按状态着色：
// normal 绿色、banned/frozen 红色、pending/inactive 黄色。
func formatUserStatusCLI(status string) string {
	if status == "" {
		return "-"
	}
	translated := i18n.APITranslate("user_status", status)
	switch strings.ToLower(status) {
	case "normal":
		return "\033[32m" + translated + "\033[0m"
	case "banned", "frozen":
		return "\033[31m" + translated + "\033[0m"
	case "pending", "inactive":
		return "\033[33m" + translated + "\033[0m"
	default:
		return translated
	}
}

// formatTimestampCLI 将 RFC3339 时间串格式化为本地可读格式；解析失败时原样返回。
func formatTimestampCLI(ts string) string {
	if ts == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

// resolveClient 根据参数构建 API 客户端：
// 优先使用命令行传入的 API Key，否则从本地配置目录读取已保存的 Key。
func resolveClient(args *config.Argument) (string, *lib.Client, error) {
	apiKey := args.ApiKey
	if apiKey == "" {
		saved, err := lib.NewSfFolder().GetKey()
		if err != nil {
			return "", nil, i18n.NewError("error.auth.not_logged_in_simple", nil, nil)
		}
		apiKey = saved
	}
	client := lib.NewClient(args.BaseDomain, apiKey)
	return apiKey, client, nil
}

// Whoami 查询并展示当前登录用户的基本信息（ID/昵称/邮箱/状态）。
func Whoami(c *cli.Context) error {
	args := parseArgument(c, meta.CmdWhoami)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := resolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	resp, err := client.UserInfoContext(c.Context)
	if err != nil {
		return cli.Exit(lib.WithStep(i18n.T("step.whoami"), err), meta.ServerError)
	}
	user := resp.Data

	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.id_label"), user.Id)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.name_label"), user.NickName)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.email_label"), user.Email)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.status_label"), formatUserStatusCLI(user.Status))
	return nil
}

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

// levelANSI 等级标识 → ANSI 真彩前景色（与 TUI levelColor 的 16 进制色一致）。
var levelANSI = map[string]string{
	"basic":    "\033[38;2;148;163;184m", // #94A3B8
	"iron":     "\033[38;2;107;114;128m", // #6B7280
	"bronze":   "\033[38;2;217;119;6m",   // #D97706
	"silver":   "\033[38;2;229;231;235m", // #E5E7EB
	"gold":     "\033[38;2;250;204;21m",  // #FACC15
	"platinum": "\033[38;2;207;250;254m", // #CFFAFE
}

// normalizeLevelKey 将 API 返回的等级名（如 "Basic"、"基础"）规范化为等级标识；无法识别时返回空字符串。
func normalizeLevelKey(name string) string {
	return levelKeyAliases[strings.ToLower(strings.TrimSpace(name))]
}

// formatLevelNameCLI 将等级名本地化并按等级着色；无法识别时原样返回。
func formatLevelNameCLI(name string) string {
	key := normalizeLevelKey(name)
	if key == "" {
		return name
	}
	translated := i18n.T("cli.plan.level_name."+key, nil)
	if strings.HasPrefix(translated, "[missing translation:") {
		translated = name
	}
	if color, ok := levelANSI[key]; ok {
		return color + translated + "\033[0m"
	}
	return translated
}

// Plan 查询当前生效套餐、下一档套餐与钱包余额信息。
func Plan(c *cli.Context) error {
	args := parseArgument(c, meta.CmdPlan)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := resolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	resp, err := client.GetPlanOverviewContext(c.Context)
	if err != nil {
		return cli.Exit(lib.WithStep(i18n.T("step.plan"), err), meta.ServerError)
	}
	data := resp.Data

	if data.Plan != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.effective_plan_label"), formatLevelNameCLI(data.Plan.EffectiveDisplayName))
		if data.Plan.NextDowngradeCheckAt != "" {
			fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.next_downgrade_label"), formatTimestampCLI(data.Plan.NextDowngradeCheckAt))
		}
	}
	if data.NextTier != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.next_tier_label"), formatLevelNameCLI(data.NextTier.DisplayName))
		if data.NextTier.NeedRechargeUsdCents > 0 {
			fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.need_recharge_label"), formatUsdCents(data.NextTier.NeedRechargeUsdCents))
		}
	}
	if data.Wallet != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.wallet.recharge_balance_label"), formatCredits(data.Wallet.RechargeBalance))
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.wallet.gift_balance_label"), formatCredits(data.Wallet.GiftBalance))
	}
	return nil
}

// Wallet 查询钱包余额：总余额，以及充值/赠送余额各自的最早到期日（剩余天数）。
func Wallet(c *cli.Context) error {
	args := parseArgument(c, meta.CmdWallet)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := resolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	walletResp, err := client.GetWalletContext(c.Context)
	if err != nil {
		return cli.Exit(lib.WithStep(i18n.T("step.wallet"), err), meta.ServerError)
	}
	wallet := walletResp.Data

	creditsResp, err := client.GetCreditsContext(c.Context, 1, 100, 0)
	if err != nil {
		return cli.Exit(lib.WithStep(i18n.T("step.credits"), err), meta.ServerError)
	}
	credits := creditsResp.Data.List

	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.credits.current_balance"), formatCredits(wallet.RechargeBalance+wallet.GiftBalance))
	printBalanceLine(i18n.T("cli.credits.recharge_balance"), formatCredits(wallet.RechargeBalance), formatExpiryCLI(earliestExpiryCLI(credits, false)))
	printBalanceLine(i18n.T("cli.credits.gift_balance"), formatCredits(wallet.GiftBalance), formatExpiryCLI(earliestExpiryCLI(credits, true)))
	return nil
}

// printBalanceLine 输出"标签: 金额"一行；有到期日时追加"日期（剩余天数）"。
func printBalanceLine(label, value, expiry string) {
	if expiry == "" {
		fmt.Fprintf(os.Stdout, "%s: %s\n", label, value)
		return
	}
	fmt.Fprintf(os.Stdout, "%s: %s %s\n", label, value, expiry)
}

// earliestExpiryCLI 返回充值/赠送积分中最早的有效期时间串，用于提示到期日。
func earliestExpiryCLI(credits []*lib.CreditItem, isGift bool) string {
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

// formatExpiryCLI 将到期时间格式化为"日期（剩余天数）"，已过期标注过期。
func formatExpiryCLI(expiredAt string) string {
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

// DayCost 查询今日按小时的消费记录：绘制每小时消费柱状图并汇总总积分与总请求数。
func DayCost(c *cli.Context) error {
	args := parseArgument(c, meta.CmdDayCost)
	setLogVerbose(args.Verbose)
	logArguments(args)

	_, client, err := resolveClient(args)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	today := time.Now().Format("2006-01-02")
	resp, err := client.GetDayCostContext(c.Context, today)
	if err != nil {
		return cli.Exit(lib.WithStep(i18n.T("step.day_cost"), err), meta.ServerError)
	}
	data := resp.Data

	if data.ByTimeResults == nil || len(data.ByTimeResults) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.day_cost.no_records"))
		return nil
	}

	times := make([]string, 0, len(data.ByTimeResults))
	creditAmounts := make([]int64, 0, len(data.ByTimeResults))
	promptQuantities := make([]int64, 0, len(data.ByTimeResults))
	var totalCredits int64
	var totalPrompts int64
	for _, r := range data.ByTimeResults {
		times = append(times, r.Time)
		creditAmounts = append(creditAmounts, r.CreditsAmount)
		promptQuantities = append(promptQuantities, r.PromptQuantity)
		totalCredits += r.CreditsAmount
		totalPrompts += r.PromptQuantity
	}

	hours := format.ExtractHourCosts(times, creditAmounts, promptQuantities)

	fmt.Fprintln(os.Stdout, i18n.T("cli.chart.intro", nil))
	fmt.Fprintln(os.Stdout)
	fmt.Fprint(os.Stdout, format.DayCostChart(hours, 40))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.day_cost.total_credits_label"), formatCredits(totalCredits))
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.day_cost.total_prompts_label"), strconv.FormatInt(totalPrompts, 10))
	return nil
}
