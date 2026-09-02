package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
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
	mainStepModelzoo
	mainStepModelzooDetail
	mainStepPriceView
	mainStepTaskModel
	mainStepAIApp
	mainStepAIAppDetail
	mainStepAIAppTask
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
	actionModelzoo    actionKind = "modelzoo"
	actionTask        actionKind = "task"
	actionAIApp       actionKind = "ai_app"
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

// myModelsResizeIdleMsg 窗口尺寸调整停止（防抖期结束）后触发列表重置
type myModelsResizeIdleMsg struct{}

// --- ModelZoo messages ---

type modelzooEndpointsDoneMsg struct {
	models []lib.ModelzooModelFlat
	err    error
}

type modelzooTagsDoneMsg struct {
	tags []lib.ModelzooTag
	err  error
}

type modelzooCategoriesDoneMsg struct {
	categories []lib.ModelzooCategoryItem
	err        error
}

type endpointDetailDoneMsg struct {
	detail *lib.ModelzooEndpointDetail
	err    error
}

type priceTableDoneMsg struct {
	pts      []lib.PriceTable
	endpoint string
	err      error
}

type taskCreatedMsg struct {
	requestID string
	err       error
}

type taskStatusMsg struct {
	status *lib.ModelZooTaskStatusResp
	err    error
}

// clearCopyFeedbackMsg 复制成功提示自动消失的消息。
type clearCopyFeedbackMsg struct{}

// --- ModelZoo filter ---

type modelzooFilterMode int

const (
	modelzooFilterNone modelzooFilterMode = iota
	modelzooFilterSeries
	modelzooFilterManufacturer
	modelzooFilterCapability
	modelzooFilterVersion
)

type modelzooInputs struct {
	search       textinput.Model
	searchActive bool

	models   []lib.ModelzooModelFlat
	filtered []lib.ModelzooModelFlat
	detail   *lib.ModelzooEndpointDetail

	tags             []lib.ModelzooTag
	categories       []lib.ModelzooCategoryItem
	seriesOpts       []filterOption
	manufacturerOpts []filterOption
	capabilityOpts   []filterOption
	versionOpts      []filterOption

	seriesFilter       string
	manufacturerFilter string
	capabilityFilter   string
	versionFilter      string

	filterMode modelzooFilterMode
	filterList list.Model
}

// --- Task model ---

type taskModelStep int

const (
	taskModelSelect taskModelStep = iota
	taskModelParamPoll
	taskModelOutputName
	taskModelPolling
	taskModelResult
)

type taskModelInputs struct {
	step       taskModelStep
	models     []lib.ModelzooModelFlat
	detail     *lib.ModelzooEndpointDetail
	params     map[string]any
	paramOrder []string
	endpoint   string

	currentParamIdx int
	requestID       string
	lastPollStatus  string
	startedAt       time.Time

	usingSelect bool
	usingTA     bool
	paramRules  []lib.ModelzooValidationRule
	paramError  error

	paramSelectList list.Model
	paramInputs     map[string]textinput.Model
	taParam         textarea.Model
	outputNameInput textinput.Model
}

// --- AI Applications ---

// aiAppFilterMode AI 应用过滤模式。
type aiAppFilterMode int

const (
	aiAppFilterNone aiAppFilterMode = iota
	aiAppFilterSort
	aiAppFilterBaseModel
)

// aiAppInputs AI 应用列表/详情状态。
type aiAppInputs struct {
	search       textinput.Model
	searchActive bool

	apps     []*lib.BizyModelInfo
	filtered []*lib.BizyModelInfo
	detail   *lib.WebAppDetail

	baseModels []string

	sortBy          string
	baseModelFilter string

	filterMode aiAppFilterMode
	filterList list.Model

	sortOpts      []filterOption
	baseModelOpts []filterOption

	enrichOffset int
}

// aiAppEnrichMsg AI 应用发布时间后台回填进度。
type aiAppEnrichMsg struct {
	done  int
	total int
	err   error
}

// aiAppTaskStep AI 应用任务步骤。
type aiAppTaskStep int

const (
	aiAppTaskParamPoll aiAppTaskStep = iota
	aiAppTaskSubmitted
	aiAppTaskPolling
	aiAppTaskResult
)

// aiAppTaskInputs AI 应用运行向导状态。
type aiAppTaskInputs struct {
	step aiAppTaskStep

	detail *lib.WebAppDetail
	fields []lib.ModelzooInputParam
	params map[string]any

	currentParamIdx int
	taskID          int64
	requestID       string
	lastPollStatus  string
	startedAt       time.Time

	usingSelect bool
	usingTA     bool
	paramRules  []lib.ModelzooValidationRule
	paramError  error

	paramSelectList list.Model
	paramInputs     map[string]textinput.Model
	taParam         textarea.Model
}

// AI 应用消息类型。
type aiAppsDoneMsg struct {
	apps  []*lib.BizyModelInfo
	total int
	err   error
}

type aiAppDetailDoneMsg struct {
	detail *lib.WebAppDetail
	err    error
}

type aiAppTaskCreatedMsg struct {
	taskID     int64
	taskStatus string
	err        error
}

type aiAppTaskStatusMsg struct {
	status *lib.ComfyTaskStatusData
	err    error
}

type aiAppTaskOutputsMsg struct {
	outputs []lib.WebAppTaskOutput
	err     error
}
