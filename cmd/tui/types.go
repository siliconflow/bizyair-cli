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
	stepIntroMethod // 选择介绍输入方式
	stepIntro
	stepPath
	stepPublic // 询问是否公开
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

type coverStatusMsg struct {
	versionIndex int
	status       string // "converting", "ready", "fallback", "done"
	message      string
}

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
	public  bool // 是否公开
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

	// 介绍输入方式：file 或 direct
	introInputMethod string

	// 介绍文件选择相关
	introPathInputFocused bool

	// 路径补全相关
	pathCompletionSuggestion string // 当前补全建议（完整路径）
	pathMatchCount           int    // 匹配项数量
}

// 打开浏览器结果消息
type openBrowserDoneMsg struct {
	msg string
	url string
	err error
}

// 基础模型类型列表加载消息
type baseModelTypesLoadedMsg struct {
	items []*lib.BaseModelTypeItem
	err   error
}
