package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDirectIntroductionIsSavedBeforeLeavingEditor(t *testing.T) {
	model := newMainModel()
	model.upStep = stepIntro
	model.act.introInputMethod = "direct"
	model.taIntro.SetValue("  TUI model introduction  ")

	model.updateUploadInputs(tea.KeyMsg{Type: tea.KeyCtrlS})

	if model.act.cur.intro != "TUI model introduction" {
		t.Fatalf("saved introduction = %q", model.act.cur.intro)
	}
	if model.upStep != stepPath {
		t.Fatalf("upload step = %v, want stepPath", model.upStep)
	}
}

func TestToActionVersionsPreservesIntroduction(t *testing.T) {
	versions := []versionItem{{
		version: "v1.0",
		base:    "SDXL",
		cover:   "/tmp/cover.png",
		intro:   "TUI model introduction",
		path:    "/tmp/model.safetensors",
		public:  true,
	}}

	got := toActionVersions(versions)
	if len(got) != 1 {
		t.Fatalf("version count = %d, want 1", len(got))
	}
	if got[0].Introduction != versions[0].intro {
		t.Fatalf("introduction = %q, want %q", got[0].Introduction, versions[0].intro)
	}
	if got[0].Version != versions[0].version || got[0].BaseModel != versions[0].base || got[0].Path != versions[0].path || got[0].CoverUrl != versions[0].cover || got[0].Public != versions[0].public {
		t.Fatalf("other version fields changed: %#v", got[0])
	}
}
