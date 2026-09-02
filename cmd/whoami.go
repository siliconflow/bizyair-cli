// 本文件实现用户信息、套餐、钱包、积分与每日消费相关的 CLI 命令，
// 以及金额/时间/状态等展示格式化辅助函数。
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func formatUserStatusCLI(status string) string {
	text := format.UserStatusText(status)
	hex := format.UserStatusHexColor(status)
	if ansi := hexToANSI(hex); ansi != "" {
		return ansi + text + "\033[0m"
	}
	return text
}

func formatLevelNameCLI(name string) string {
	text := format.LevelDisplayName(name)
	hex := format.LevelHexColor(name)
	if ansi := hexToANSI(hex); ansi != "" {
		return ansi + text + "\033[0m"
	}
	return text
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

	result := actions.WhoamiContext(c.Context, client)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}
	if result.User == nil {
		return cli.Exit(i18n.NewError("error.auth.not_logged_in_simple", nil, nil), meta.LoadError)
	}
	user := result.User

	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.id_label"), user.Id)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.name_label"), user.NickName)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.email_label"), user.Email)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.whoami.status_label"), formatUserStatusCLI(user.Status))
	return nil
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

	result := actions.PlanContext(c.Context, client)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}
	if result.Plan == nil {
		return cli.Exit(i18n.NewError("error.auth.not_logged_in_simple", nil, nil), meta.LoadError)
	}
	data := result.Plan

	if data.Plan != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.effective_plan_label"), formatLevelNameCLI(data.Plan.EffectiveDisplayName))
		if data.Plan.NextDowngradeCheckAt != "" {
			fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.next_downgrade_label"), format.FormatTimestamp(data.Plan.NextDowngradeCheckAt))
		}
	}
	if data.NextTier != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.next_tier_label"), formatLevelNameCLI(data.NextTier.DisplayName))
		if data.NextTier.NeedRechargeUsdCents > 0 {
			fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.plan.need_recharge_label"), format.FormatUsdCents(data.NextTier.NeedRechargeUsdCents))
		}
	}
	if data.Wallet != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.wallet.recharge_balance_label"), format.FormatCredits(data.Wallet.RechargeBalance))
		fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.wallet.gift_balance_label"), format.FormatCredits(data.Wallet.GiftBalance))
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

	result := actions.CreditsContext(c.Context, client)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.credits.current_balance"), format.FormatCredits(result.TotalAmount))
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.credits.recharge_balance"), format.FormatExpiryLine(result.Credits, false, format.FormatCredits(result.RechargeAmount)))
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.credits.gift_balance"), format.FormatExpiryLine(result.Credits, true, format.FormatCredits(result.GiftAmount)))
	return nil
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

	result := actions.DayCostContext(c.Context, client)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}
	if len(result.Records) == 0 {
		fmt.Fprintln(os.Stdout, i18n.T("cli.day_cost.no_records"))
		return nil
	}

	hours, totalCredits, totalPrompts := format.SummarizeDayCost(result.Records)

	fmt.Fprintln(os.Stdout, i18n.T("cli.chart.intro", nil))
	fmt.Fprintln(os.Stdout)
	fmt.Fprint(os.Stdout, format.DayCostChart(hours, 40))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.day_cost.total_credits_label"), format.FormatCredits(totalCredits))
	fmt.Fprintf(os.Stdout, "%s: %s\n", i18n.T("cli.day_cost.total_prompts_label"), strconv.FormatInt(totalPrompts, 10))
	return nil
}
