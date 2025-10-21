package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
)

// checkModelExists 检查模型名是否已存在
func checkModelExists(apiKey, modelName, modelType string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		exists, err := client.CheckModelExists(modelName, modelType)
		return checkModelExistsDoneMsg{exists: exists, err: err}
	}
}

// loadBaseModelTypes 从后端加载基础模型类型列表
func loadBaseModelTypes(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		resp, err := client.GetBaseModelTypes()
		if err != nil {
			return baseModelTypesLoadedMsg{items: nil, err: err}
		}
		return baseModelTypesLoadedMsg{items: resp.Data, err: nil}
	}
}

// 多版本上传
func runUploadActionMulti(u uploadInputs, versions []versionItem) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan tea.Msg, 64)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			defer close(ch)

			// 获取API Key
			apiKey, err := lib.NewSfFolder().GetKey()
			if err != nil || apiKey == "" {
				ch <- actionDoneMsg{
					out: "",
					err: lib.WithStep("上传/鉴权", fmt.Errorf("未登录或缺少 API Key，请先登录")),
				}
				return
			}

			// 准备版本输入参数
			actionVersions := make([]actions.VersionInput, len(versions))
			for i, v := range versions {
				actionVersions[i] = actions.VersionInput{
					Version:      v.version,
					Path:         v.path,
					BaseModel:    v.base,
					Introduction: v.intro,
					CoverUrl:     v.cover,
					Public:       v.public,
				}
			}

			// 准备上传输入参数
			input := actions.UploadInput{
				ApiKey:     apiKey,
				BaseDomain: meta.DefaultDomain,
				ModelType:  u.typ,
				ModelName:  u.name,
				Versions:   actionVersions,
				Overwrite:  false,
				Context:    ctx, // 传递可取消的context
			}

			// 创建TUI回调
			callback := &tuiUploadCallback{ch: ch}

			// 发送开始消息（带上真正的cancel函数）
			ch <- uploadStartMsg{ch: ch, cancel: cancel}

			// 执行上传
			result := actions.ExecuteUpload(input, callback)

			// 处理结果
			if !result.Success {
				if result.CanceledByUser {
					var sb strings.Builder
					sb.WriteString("✓ 上传已取消\n\n")
					sb.WriteString("已上传的部分已保存checkpoint，下次上传相同文件时会自动续传。\n")
					folder, _ := lib.GetCheckpointDir()
					if folder != "" {
						sb.WriteString(fmt.Sprintf("Checkpoint文件位置: %s\n", folder))
					}
					ch <- actionDoneMsg{out: sb.String(), err: nil}
					return
				}

				var sb strings.Builder
				sb.WriteString("上传失败\n")
				for _, err := range result.Errors {
					sb.WriteString(fmt.Sprintf("- %v\n", err))
				}
				ch <- actionDoneMsg{
					out: sb.String(),
					err: lib.WithStep("上传", fmt.Errorf("上传失败")),
				}
				return
			}

			// 成功
			var out strings.Builder
			out.WriteString("Uploaded successfully\n")
			if result.SuccessCount < result.TotalCount {
				out.WriteString(fmt.Sprintf("部分版本失败：成功 %d/%d\n",
					result.SuccessCount, result.TotalCount))
				for _, err := range result.Errors {
					out.WriteString(fmt.Sprintf("- %v\n", err))
				}
			}
			ch <- actionDoneMsg{out: out.String(), err: nil}
		}()

		return uploadStartMsg{ch: ch, cancel: cancel}
	}
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
