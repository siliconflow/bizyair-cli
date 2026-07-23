package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// checkModelExists 检查模型名是否已存在
func checkModelExists(ctx context.Context, baseDomain, apiKey, modelName, modelType string) tea.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		client := lib.NewClient(baseDomain, apiKey)
		exists, err := client.CheckModelExistsContext(ctx, modelName, modelType)
		return checkModelExistsDoneMsg{exists: exists, err: err}
	}
}

// loadBaseModelTypes 从后端加载基础模型类型列表
func loadBaseModelTypes(ctx context.Context, baseDomain, apiKey string) tea.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		client := lib.NewClient(baseDomain, apiKey)
		resp, err := client.GetBaseModelTypesContext(ctx)
		if err != nil {
			return baseModelTypesLoadedMsg{items: nil, err: err}
		}
		return baseModelTypesLoadedMsg{items: resp.Data, err: nil}
	}
}

// 多版本上传
func runUploadActionMulti(parentCtx context.Context, baseDomain string, u uploadInputs, versions []versionItem) tea.Cmd {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	return func() tea.Msg {
		ch := make(chan tea.Msg, 64)
		ctx, cancel := context.WithCancel(parentCtx)

		go func() {
			defer close(ch)

			// 获取API Key
			apiKey, err := lib.NewSfFolder().GetKey()
			if err != nil || apiKey == "" {
				ch <- actionDoneMsg{
					out: "",
					err: lib.WithStep(i18n.T("step.login_validation", nil), i18n.NewError("error.auth.api_key_missing", nil, nil)),
				}
				return
			}

			// 创建客户端
			client := lib.NewClient(baseDomain, apiKey)

			// 从API获取基础模型列表
			allowedModels := []string{}
			resp, err := client.GetBaseModelTypesContext(ctx)
			if err != nil {
				ch <- actionDoneMsg{
					out: "",
					err: lib.WithStep(i18n.T("step.get_base_models", nil), i18n.NewError("error.base_models.fetch_failed", nil, err)),
				}
				return
			}
			if resp.Data != nil {
				for _, item := range resp.Data {
					allowedModels = append(allowedModels, item.Value)
				}
			}
			if len(allowedModels) == 0 {
				ch <- actionDoneMsg{
					out: "",
					err: lib.WithStep(i18n.T("step.get_base_models", nil), i18n.NewError("error.base_models.empty", nil, nil)),
				}
				return
			}

			// 准备版本输入参数
			actionVersions := toActionVersions(versions)

			// 准备上传输入参数
			input := actions.UploadInput{
				ApiKey:            apiKey,
				BaseDomain:        baseDomain,
				ModelType:         u.typ,
				ModelName:         u.name,
				Versions:          actionVersions,
				Overwrite:         false,
				Context:           ctx,           // 传递可取消的context
				AllowedBaseModels: allowedModels, // 传入从API获取的列表
			}

			// 创建TUI回调
			callback := &tuiUploadCallback{ch: ch}

			// 发送开始消息（带上真正的cancel函数）
			ch <- uploadStartMsg{ch: ch, cancel: cancel}

			// 执行上传
			api := lib.NewClient(baseDomain, apiKey)
			result := actions.ExecuteUpload(api, input, callback)

			// 处理结果
			if !result.Success {
				if result.CanceledByUser {
					var sb strings.Builder
					sb.WriteString(i18n.T("tui.upload.canceled_title", nil) + "\n\n")
					sb.WriteString(i18n.T("tui.upload.checkpoint_saved", nil) + "\n")
					folder, _ := lib.GetCheckpointDir()
					if folder != "" {
						sb.WriteString(i18n.T("tui.upload.checkpoint_location", map[string]any{"Path": folder}) + "\n")
					}
					ch <- actionDoneMsg{out: sb.String(), err: nil}
					return
				}

				var sb strings.Builder
				sb.WriteString(i18n.T("tui.upload.failed_title", nil) + "\n")
				for _, err := range result.Errors {
					sb.WriteString(fmt.Sprintf("- %v\n", err))
				}
				ch <- actionDoneMsg{
					out: sb.String(),
					err: lib.WithStep(i18n.T("step.upload", nil), i18n.NewError("cli.upload.failed", nil, nil)),
				}
				return
			}

			// 成功
			var out strings.Builder
			out.WriteString(i18n.T("tui.upload.success", nil) + "\n")
			if result.SuccessCount < result.TotalCount {
				out.WriteString(i18n.T("tui.upload.partial", map[string]any{
					"Success": result.SuccessCount,
					"Total":   result.TotalCount,
				}) + "\n")
				for _, err := range result.Errors {
					out.WriteString(fmt.Sprintf("- %v\n", err))
				}
			}
			ch <- actionDoneMsg{out: out.String(), err: nil}
		}()

		return uploadStartMsg{ch: ch, cancel: cancel}
	}
}

func toActionVersions(versions []versionItem) []actions.VersionInput {
	actionVersions := make([]actions.VersionInput, len(versions))
	for i, version := range versions {
		actionVersions[i] = actions.VersionInput{
			Version:      version.version,
			Path:         version.path,
			BaseModel:    version.base,
			Introduction: version.intro,
			CoverUrl:     version.cover,
			Public:       version.public,
		}
	}
	return actionVersions
}

// tuiUploadCallback TUI的进度回调实现
type tuiUploadCallback struct {
	ch chan<- tea.Msg
}

func (t *tuiUploadCallback) OnProgress(progress actions.UploadProgress) {
	select {
	case t.ch <- uploadProgMsg{
		fileIndex: fmt.Sprintf("%d/%d", progress.VersionIndex+1, progress.VersionTotal),
		fileName:  progress.FileName,
		consumed:  progress.Consumed,
		total:     progress.Total,
		verIdx:    progress.VersionIndex,
	}:
	default:
	}
}

func (t *tuiUploadCallback) OnVersionStart(index, total int, fileName string) {
	// TUI可以选择不实现这个回调，或者发送特定消息
}

func (t *tuiUploadCallback) OnVersionComplete(index, total int, fileName string, err error) {
	// TUI可以选择不实现这个回调，或者发送特定消息
}

func (t *tuiUploadCallback) OnCoverStatus(index, total int, status, message string) {
	select {
	case t.ch <- coverStatusMsg{
		versionIndex: index,
		status:       status,
		message:      message,
	}:
	default:
	}
}
