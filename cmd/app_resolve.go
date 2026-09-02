package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/urfave/cli/v2"
)

// webAppVersionRef 表示解析后的 AI 应用：应用本身与其可用的版本 ID。
type webAppVersionRef struct {
	App       *lib.BizyModelInfo
	VersionID int64
}

// resolveWebAppVersion 将 <名称或版本ID> 参数解析为 AI 应用与版本 ID：
//   - 纯数字按版本 ID 直查（保持兼容），不返回应用列表项；
//   - 否则按名称调用社区列表精确匹配（大小写不敏感）；
//   - 多个同名时返回候选列表，由调用方提示用户区分。
func resolveWebAppVersion(c *cli.Context, client *lib.Client, args *config.Argument, arg string) (*webAppVersionRef, error) {
	if versionID, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64); err == nil {
		return &webAppVersionRef{VersionID: versionID}, nil
	}

	result := actions.ResolveAIAppByName(c.Context, client, arg)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(result.Candidates) > 0 {
		return nil, ambiguousAppNameError(result.Candidates)
	}
	app := result.App
	if app == nil || len(app.Versions) == 0 || app.Versions[0] == nil {
		return nil, i18n.NewError("cli.app.not_found", map[string]any{"ID": arg}, nil)
	}
	return &webAppVersionRef{App: app, VersionID: app.Versions[0].Id}, nil
}

// ambiguousAppNameError 生成名称不唯一的错误提示，并列出候选应用。
func ambiguousAppNameError(candidates []*lib.BizyModelInfo) error {
	name := ""
	if len(candidates) > 0 {
		name = candidates[0].Name
	}
	lines := make([]string, 0, len(candidates))
	for _, app := range candidates {
		if app == nil {
			continue
		}
		lines = append(lines, i18n.T("cli.app.candidates_line", map[string]any{
			"Name":      lib.DisplayOrDash(app.Name),
			"BaseModel": lib.DisplayOrDash(lib.AIApplicationBaseModel(app)),
			"ID":        app.Id,
		}))
	}
	header := i18n.T("cli.app.ambiguous", map[string]any{"Name": name})
	return fmt.Errorf("%s\n%s", header, strings.Join(lines, "\n"))
}
