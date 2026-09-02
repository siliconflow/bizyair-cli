package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// fetchAIApps 异步拉取社区 AI 应用列表。
func fetchAIApps(api lib.BizyAPI, keyword string) tea.Cmd {
	return func() tea.Msg {
		result := actions.ListAIApplications(context.Background(), api, keyword, lib.AIAppSortRecent, nil, 1, 500)
		return aiAppsDoneMsg{apps: result.Apps, total: result.Total, err: result.Error}
	}
}

// fetchAIAppDetail 异步拉取 AI 应用详情（版本详情 + 工作流详情合并）。
func fetchAIAppDetail(api lib.BizyAPI, versionID int64) tea.Cmd {
	return func() tea.Msg {
		result := actions.GetAIAppDetail(context.Background(), api, versionID)
		return aiAppDetailDoneMsg{detail: result.Detail, err: result.Error}
	}
}

// createAIAppTask 异步创建 AI 应用任务（/v1/webapp/task/create，返回 task_id 轮询）。
func createAIAppTask(api lib.BizyAPI, req lib.WebAppTaskCreateReq) tea.Cmd {
	return func() tea.Msg {
		result := actions.CreateWebAppTask(context.Background(), api, req)
		return aiAppTaskCreatedMsg{
			taskID:     result.TaskID,
			taskStatus: result.TaskStatus,
			wssURL:     result.WssURL,
			err:        result.Error,
		}
	}
}

// pollAIAppTaskStatus 按 Comfy task_id 轮询 AI 应用任务状态。
func pollAIAppTaskStatus(api lib.BizyAPI, taskID int64) tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		result := actions.GetWebAppTaskStatus(context.Background(), api, taskID)
		return aiAppTaskStatusMsg{status: result.Status, err: result.Error}
	})
}

// getAIAppTaskOutputs 异步获取 AI 应用任务输出（按 request_id）。
func getAIAppTaskOutputs(api lib.BizyAPI, requestID string) tea.Cmd {
	return func() tea.Msg {
		result := actions.GetWebAppTaskOutputs(context.Background(), api, requestID)
		return aiAppTaskOutputsMsg{outputs: result.Outputs, err: result.Error}
	}
}

// cancelAIAppTask 异步取消排队中的 AI 应用任务（按 request_id）。
func cancelAIAppTask(api lib.BizyAPI, requestID string) tea.Cmd {
	return func() tea.Msg {
		err := actions.CancelWebAppTask(context.Background(), api, requestID)
		return aiAppTaskCancelledMsg{err: err}
	}
}