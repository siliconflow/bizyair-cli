package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

func TestTUIModelsAreLocalized(t *testing.T) {
	tests := []struct {
		lang i18n.Language
		want []string
	}{
		{lang: i18n.English, want: []string{"Upload", "Model type"}},
		{lang: i18n.SimplifiedChinese, want: []string{"上传", "选择模型类型"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			if err := i18n.Configure(tt.lang); err != nil {
				t.Fatal(err)
			}
			model := newMainModel()
			model.currentAction = actionUpload
			model.step = mainStepAction
			model.upStep = stepType
			view := model.renderUploadStepsView()
			for _, want := range tt.want {
				if !strings.Contains(view, want) {
					t.Fatalf("upload view does not contain %q:\n%s", want, view)
				}
			}
		})
	}
}

func TestTUIViewDoesNotExceedNarrowTerminal(t *testing.T) {
	if err := i18n.Configure(i18n.English); err != nil {
		t.Fatal(err)
	}
	model := newMainModel()
	model.step = mainStepLogin
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 42, Height: 20})
	model = updated.(mainModel)

	for _, line := range strings.Split(model.View(), "\n") {
		if width := lipgloss.Width(line); width > 42 {
			t.Fatalf("rendered line width %d exceeds terminal width 42: %q", width, line)
		}
	}
}
