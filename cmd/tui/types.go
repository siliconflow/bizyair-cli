package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
	mainStepUserInfo
	mainStepMyModelsList
	mainStepModelDetail
)

type actionKind string

const (
	actionUpload      actionKind = "upload"
	actionLsModel     actionKind = "ls_model"
	actionModelDetail actionKind = "model_detail"
	actionLogout      actionKind = "logout"
	actionExit        actionKind = "exit"
	actionUserInfo    actionKind = "user_info"
	actionWhoami      actionKind = "whoami"
	actionPlan        actionKind = "plan"
	actionWallet      actionKind = "wallet"
	actionCredits     actionKind = "credits"
	actionDayCost     actionKind = "day_cost"
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

// errStep 从错误中提取步骤信息
func errStep(err error) string {
	return lib.GetStep(err)
}

// 列表项（与 bubbles/list 兼容）
type listItem struct{ title, desc, value string }

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

// userInfoInputs 用户信息子菜单状态
type userInfoInputs struct {
	selectedAction actionKind
}

// 基础模型类型列表加载消息
type baseModelTypesLoadedMsg struct {
	items []*lib.BaseModelTypeItem
	err   error
}

// --- User info messages ---

// whoamiDoneMsg 用户信息查询结果消息。
type whoamiDoneMsg struct {
	user *lib.UserInfo
	err  error
}

// planDoneMsg 套餐查询结果消息。
type planDoneMsg struct {
	plan *lib.PlanOverviewResp
	err  error
}

// creditsDoneMsg 积分明细查询结果消息。
type creditsDoneMsg struct {
	credits     []*lib.CreditItem
	total       int
	giftAmt     int64
	rechargeAmt int64
	totalAmt    int64
	err         error
}

// dayCostDoneMsg 每日消费查询结果消息。
type dayCostDoneMsg struct {
	records []*lib.CostByTimeResult
	err     error
}

// myModelsFilterMode 筛选选择器类型
type myModelsFilterMode int

const (
	myModelsFilterNone myModelsFilterMode = iota
	myModelsFilterType
	myModelsFilterSort
	myModelsFilterBaseModel
)

// filterOption 筛选选项
type filterOption struct {
	label string
	value string
	count int
}

// filterBarItem 筛选条片段
type filterBarItem struct {
	label  string
	value  string
	active bool
}

// myModelsInputs 我的模型列表页输入状态
type myModelsInputs struct {
	typeFilter      string
	sortBy          string
	baseModelFilter string
	filterMode      myModelsFilterMode
	filterList      list.Model

	typeOpts []filterOption
	sortOpts []filterOption
	bmOpts   []filterOption

	search       textinput.Model
	searchActive bool
}

// myModelsDoneMsg 我的模型列表加载完成消息
type myModelsDoneMsg struct {
	models []*lib.BizyModelInfo
	total  int
	err    error
}

// modelDetailDoneMsg 模型详情加载完成消息
type modelDetailDoneMsg struct {
	detail *lib.BizyModelDetail
	err    error
}

// modelDeletedMsg 模型删除完成消息
type modelDeletedMsg struct {
	success bool
	err     error
}

// modelPublicToggledMsg 模型公开状态切换完成消息
type modelPublicToggledMsg struct {
	success bool
	err     error
}
