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
		lang            i18n.Language
		menuTitle       string
		namePlaceholder string
		typeTitle       string
		stepText        []string
		doneTitle       string
	}{
		{
			lang: i18n.English, menuTitle: "Choose an action", namePlaceholder: "Enter a model name", typeTitle: "Upload · Step 1/11 · Model type", doneTitle: "Completed",
			stepText: []string{"Model type", "Model name", "Version name", "Base Model", "Cover upload method", "Cover URL", "Introduction input method", "Model introduction", "Model file", "Public visibility", "More versions", "Confirm all versions"},
		},
		{
			lang: i18n.SimplifiedChinese, menuTitle: "请选择功能", namePlaceholder: "请输入模型名称", typeTitle: "上传 · 第 1/11 步 · 选择模型类型", doneTitle: "执行完成",
			stepText: []string{"选择模型类型", "模型名称", "版本名称", "Base Model", "选择封面上传方式", "输入封面 URL", "选择介绍输入方式", "模型介绍", "选择模型文件", "是否公开此版本", "继续添加版本", "确认所有版本"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			if err := i18n.Configure(tt.lang); err != nil {
				t.Fatal(err)
			}
			model := newMainModel()
			if model.menu.Title != tt.menuTitle {
				t.Fatalf("menu title = %q, want %q", model.menu.Title, tt.menuTitle)
			}
			if !strings.Contains(model.inpName.Placeholder, tt.namePlaceholder) {
				t.Fatalf("name placeholder = %q", model.inpName.Placeholder)
			}
			model.currentAction = actionUpload
			model.step = mainStepAction
			model.act.coverUploadMethod = "url"
			model.act.introInputMethod = "direct"
			model.act.versions = []versionItem{{version: "v1.0", base: "SDXL", cover: "cover.jpg", intro: "intro", path: "model.safetensors"}}
			model.act.cur = model.act.versions[0]
			steps := []uploadStep{stepType, stepName, stepVersion, stepBase, stepCoverMethod, stepCover, stepIntroMethod, stepIntro, stepPath, stepPublic, stepAskMore, stepConfirm}
			for index, step := range steps {
				model.upStep = step
				if view := model.renderUploadStepsView(); !strings.Contains(view, tt.stepText[index]) {
					t.Errorf("step %d does not contain %q:\n%s", step, tt.stepText[index], view)
				}
			}
			model.upStep = stepType
			if view := model.renderUploadStepsView(); !strings.Contains(view, tt.typeTitle) {
				t.Fatalf("upload view does not contain %q:\n%s", tt.typeTitle, view)
			}
			model.width, model.height = 80, 30
			model.step = mainStepOutput
			model.err = nil
			model.output = "result"
			if view := model.View(); !strings.Contains(view, tt.doneTitle) {
				t.Fatalf("output page does not contain %q", tt.doneTitle)
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
