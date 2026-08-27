package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

func fetchModelzooEndpoints(api lib.BizyAPI, keyword, billingUnit, sort string, showDeprecated bool) tea.Cmd {
	return func() tea.Msg {
		result := actions.ListModelzooEndpoints(context.Background(), api, keyword, billingUnit, sort, showDeprecated)
		return modelzooEndpointsDoneMsg{models: result.Models, err: result.Error}
	}
}

func fetchModelzooTags(api lib.BizyAPI) tea.Cmd {
	return func() tea.Msg {
		result := actions.ListModelzooTags(context.Background(), api)
		return modelzooTagsDoneMsg{tags: result.Tags, err: result.Error}
	}
}

func fetchEndpointDetail(api lib.BizyAPI, endpoint string) tea.Cmd {
	return func() tea.Msg {
		result := actions.GetEndpointDetail(context.Background(), api, endpoint)
		return endpointDetailDoneMsg{detail: result.Detail, err: result.Error}
	}
}

func createTaskCmd(api lib.BizyAPI, endpoint string, params map[string]any) tea.Cmd {
	return func() tea.Msg {
		result := actions.CreateTask(context.Background(), api, endpoint, params)
		return taskCreatedMsg{requestID: result.RequestID, err: result.Error}
	}
}

func cancelTaskCmd(api lib.BizyAPI, requestID string) tea.Cmd {
	return func() tea.Msg {
		err := actions.GetTaskCancel(context.Background(), api, requestID)
		return taskCancelledMsg{err: err}
	}
}

func pollTaskStatus(api lib.BizyAPI, requestID string) tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		result := actions.GetTaskStatus(context.Background(), api, requestID)
		return taskStatusMsg{status: result.Status, err: result.Error}
	})
}
