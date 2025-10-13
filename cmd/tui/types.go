package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
)

// 菜单/流程步骤
type mainStep int

const (
	mainStepHome mainStep = iota
	mainStepLogin
	mainStepMenu
	mainStepAction
	mainStepOutput
)

type actionKind string

const (
	actionUpload  actionKind = "upload"
	actionLsModel actionKind = "ls_model"
	actionWhoami  actionKind = "whoami"
	actionLogout  actionKind = "logout"
	actionExit    actionKind = "exit"
)

// 上传步骤
type uploadStep int

const (
	stepType uploadStep = iota
	stepName
	stepVersion
	stepBase
	stepCover
	stepIntro
	stepPath
	stepAskMore
	stepConfirm
)

// 消息类型
type loginDoneMsg struct {
	ok  bool
	err error
}

type actionDoneMsg struct {
	out string
	err error
}

type uploadStartMsg struct {
	ch     <-chan tea.Msg
	cancel func()
}

type uploadProgMsg struct {
	fileIndex string
	fileName  string
	consumed  int64
	total     int64
	verIdx    int
}

type uploadCancelMsg struct{}
type clearFilePickerErrorMsg struct{}

type checkModelExistsDoneMsg struct {
	exists bool
	err    error
}

// 带步骤信息的错误
type stepError struct {
	Step string
	Err  error
}

func (e *stepError) Error() string { return e.Err.Error() }

func withStep(step string, err error) error {
	if err == nil {
		return nil
	}
	return &stepError{Step: step, Err: err}
}

func errStep(err error) string {
	var se *stepError
	if errors.As(err, &se) {
		return se.Step
	}
	return ""
}

// 列表项（与 bubbles/list 兼容）
type listItem struct{ title, desc string }

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

// 菜单项
type menuEntry struct {
	listItem
	key actionKind
}

// 上传所需输入
type uploadInputs struct {
	typ  string
	name string
}

// 单个版本输入
type versionItem struct {
	version string
	base    string
	cover   string
	intro   string
	path    string
}

// 动作输入状态
type actionInputs struct {
	// 通用
	confirming bool

	// Upload
	u uploadInputs
	// 多版本
	versions []versionItem
	cur      versionItem

	// Filepicker state
	useFilePicker bool
	filePickerErr error

	// 路径输入相关
	pathInputFocused bool
}

// 列表消息
type modelListLoadedMsg struct {
	models []*lib.BizyModelInfo
	total  int
	err    error
}

// 基础模型类型列表加载消息
type baseModelTypesLoadedMsg struct {
	items []*lib.BaseModelTypeItem
	err   error
}
