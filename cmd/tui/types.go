package tui

import (
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
	stepCoverMethod // 选择封面上传方式
	stepCover       // 实际上传封面
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

// withStep 使用 lib 层的错误处理
func withStep(step string, err error) error {
	return lib.WithStep(step, err)
}

// errStep 从错误中提取步骤信息
func errStep(err error) string {
	return lib.GetStep(err)
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

	// 封面上传方式：url 或 local
	coverUploadMethod string
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
