package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

type mainModel struct {
	step          mainStep
	loggedIn      bool
	apiKey        string
	err           error
	currentAction actionKind

	width  int
	height int

	menu            list.Model
	inpApi          textinput.Model
	typeList        list.Model
	baseList        list.Model
	coverMethodList list.Model
	introMethodList list.Model
	moreList        list.Model
	inpName         textinput.Model
	inpPath         textinput.Model
	inpCover        textinput.Model
	inpExt          textinput.Model
	inpVersion      textinput.Model
	taIntro         textarea.Model

	filepicker   filepicker.Model
	selectedFile string

	modelTable       table.Model
	modelList        []*lib.BizyModelInfo
	modelListTotal   int
	loadingModelList bool
	nameColWidth     int
	fileColWidth     int

	viewingModelDetail bool
	modelDetail        *lib.BizyModelDetail
	loadingModelDetail bool
	detailViewport     viewport.Model
	detailContent      string

	confirmingDelete bool
	deleteTargetId   int64
	deleteTargetName string

	confirmingExit bool

	act    actionInputs
	upStep uploadStep

	running     bool
	canceling   bool
	output      string
	sp          spinner.Model
	progress    progress.Model
	verProgress []progress.Model
	verConsumed []int64
	verTotal    []int64

	titleStyle lipgloss.Style
	hintStyle  lipgloss.Style
	panelStyle lipgloss.Style
	btnStyle   lipgloss.Style

	framePadX int
	framePadY int

	logo           string
	logoStyle      lipgloss.Style
	smallLogo      string
	smallLogoStyle lipgloss.Style

	uploadCh   <-chan tea.Msg
	uploadProg uploadProgMsg
	cancelFn   func()

	coverUrlInputFocused  bool
	coverPathInputFocused bool

	loadingBaseModelTypes bool
	baseModelTypes        []*lib.BaseModelTypeItem

	// 上传速率追踪
	lastUploadTime     time.Time // 上次更新进度的时间
	lastUploadBytes    int64     // 上次更新时的字节数
	currentUploadSpeed int64     // 当前上传速率（字节/秒）

	// 多版本上传的速率追踪
	verLastTime  []time.Time
	verLastBytes []int64
	verSpeed     []int64

	// 封面状态追踪
	coverStatus        string // 当前封面状态信息
	coverStatusWarning bool   // 是否为警告状态（如转换失败回退）
}

func newMainModel() mainModel {
	mItems := []list.Item{
		menuEntry{listItem{title: "上传模型", desc: "交互式收集参数并上传"}, actionUpload},
		menuEntry{listItem{title: "我的模型", desc: "浏览我的模型"}, actionLsModel},
		menuEntry{listItem{title: "当前账户信息", desc: "显示 whoami"}, actionWhoami},
		menuEntry{listItem{title: "退出登录", desc: "清除本地 API Key"}, actionLogout},
		menuEntry{listItem{title: "退出程序", desc: "离开 BizyAir CLI"}, actionExit},
	}
	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#04B575")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)
	menuList := list.New(mItems, d, 30, len(mItems)*8)
	menuList.Title = "请选择功能"
	menuList.SetShowStatusBar(false)
	menuList.SetShowPagination(false)

	tItems := make([]list.Item, 0, len(meta.ModelTypes))
	for _, t := range meta.ModelTypes {
		s := string(t)
		tItems = append(tItems, listItem{title: s})
	}
	tp := list.New(tItems, d, 30, len(tItems))
	tp.Title = "选择模型类型"
	tp.SetShowStatusBar(false)
	tp.SetShowPagination(false)

	// 初始化时创建空的基础模型列表，将在进入上传步骤时从后端加载
	bl := list.New([]list.Item{}, d, 30, 12)
	bl.Title = "选择 Base Model（必选）"

	coverMethodItems := []list.Item{
		listItem{title: "通过 URL 上传", desc: "输入图片或视频的网络链接"},
		listItem{title: "从本地上传", desc: "从本地文件系统选择文件"},
	}
	cml := list.New(coverMethodItems, d, 30, 12)
	cml.Title = "选择封面上传方式"
	cml.SetShowStatusBar(false)
	cml.SetShowPagination(false)

	introMethodItems := []list.Item{
		listItem{title: "通过文件导入", desc: "从 .txt 或 .md 文件导入介绍内容"},
		listItem{title: "直接输入", desc: "在文本编辑器中直接输入"},
	}
	iml := list.New(introMethodItems, d, 30, 12)
	iml.Title = "选择介绍输入方式"
	iml.SetShowStatusBar(false)
	iml.SetShowPagination(false)

	moreItems := []list.Item{listItem{title: "是，继续添加版本"}, listItem{title: "否，进入确认"}}
	ml := list.New(moreItems, d, 30, 12)
	ml.Title = "是否继续添加版本？"
	ml.SetShowStatusBar(false)
	ml.SetShowPagination(false)

	inApi := textinput.New()
	inApi.Placeholder = "请输入 API Key"
	inName := textinput.New()
	inName.Placeholder = "请输入模型名称（字母/数字/下划线/短横线）"
	inVer := textinput.New()
	inVer.Placeholder = "请输入版本名称（默认: v1.0）"
	taIntro := textarea.New()
	taIntro.Placeholder = "输入模型介绍（最多5000字，Ctrl+D 提交）"
	taIntro.CharLimit = 5000
	taIntro.SetHeight(20)
	taIntro.ShowLineNumbers = false
	inPath := textinput.New()
	inPath.Placeholder = "请输入文件路径（仅文件）"
	if homeDir, err := os.UserHomeDir(); err == nil {
		inPath.SetValue(homeDir + "/")
	}
	inCover := textinput.New()
	inCover.Placeholder = "可选，多地址以 ; 分隔"
	inExt := textinput.New()
	inExt.Placeholder = "可选，文件扩展名（如 .safetensors）"

	fp := filepicker.New()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	fp.CurrentDirectory = homeDir
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.SetHeight(10)
	fp.AutoHeight = false
	fp.Styles.EmptyDirectory = fp.Styles.EmptyDirectory.SetString("此目录为空。使用方向键导航到其他目录。注意文件路径输入框中的路径与下面文件选择器中的内容的同步")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	pr := progress.New(progress.WithDefaultGradient())

	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "名称", Width: 25},
		{Title: "类型", Width: 12},
		{Title: "版本数", Width: 8},
		{Title: "基础模型", Width: 15},
		{Title: "文件名", Width: 60},
	}
	modelTable := table.New(table.WithColumns(columns), table.WithFocused(true), table.WithHeight(10))
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#36A3F7"))
	s.Cell = s.Cell.PaddingLeft(0).PaddingRight(0)
	s.Header = s.Header.PaddingLeft(0).PaddingRight(0)
	s.Selected = s.Selected.PaddingLeft(0).PaddingRight(0)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)
	modelTable.SetStyles(s)

	// 初始化模型详情 viewport
	detailViewport := viewport.New(80, 20)
	detailViewport.Style = lipgloss.NewStyle()

	m := mainModel{
		step:            mainStepHome,
		menu:            menuList,
		inpApi:          inApi,
		typeList:        tp,
		baseList:        bl,
		coverMethodList: cml,
		introMethodList: iml,
		moreList:        ml,
		inpName:         inName,
		inpPath:         inPath,
		inpCover:        inCover,
		inpExt:          inExt,
		inpVersion:      inVer,
		taIntro:         taIntro,
		filepicker:      fp,
		modelTable:      modelTable,
		detailViewport:  detailViewport,
		titleStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7")),
		hintStyle:       lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		panelStyle:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2),
		btnStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#04B575")).Padding(0, 1).Bold(true),
		sp:              sp,
		progress:        pr,
		logo: strings.Join([]string{
			` .-. .-')              .-') _              ('-.             _  .-')   `,
			` \  ( OO )            (  OO) )            ( OO ).-.        ( \( -O )  `,
			`  ;-----.\   ,-.-') ,(_)----.  ,--.   ,--./ . --. /  ,-.-') ,------.  `,
			`  | .-.  |   |  |OO)|       |   \  ` + "`" + `.'  / | \-.  \   |  |OO)|   /` + "`" + `. ' `,
			`  | '-' /_)  |  |  \'--.   /  .-')     /.-'-'  |  |  |  |  \|  /  | | `,
			`  | .-. ` + "`" + `.   |  |(_/(_/   /  (OO  \   /  \| |_.'  |  |  |(_/|  |_.' | `,
			`  | |  \  | ,|  |_.' /   /___ |   /  /\_  |  .-.  | ,|  |_.'|  .  '.' `,
			`  | '--'  /(_|  |   |        |` + "`" + `-./  /.__) |  | |  |(_|  |   |  |\  \  `,
			`  ` + "`" + `------'   ` + "`" + `--'   ` + "`" + `--------'  ` + "`" + `--'      ` + "`" + `--' ` + "`" + `--'  ` + "`" + `--'   ` + "`" + `--' '--' `,
		}, "\n"),
		logoStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")),
		smallLogo: strings.Join([]string{
			".----. .-. .---..-.  .-..--.  .-..----.    ",
			"| {}  }| |{_   / \\ \\/ // {} \\ | || {}  }   ",
			"| {}  }| | /    } }  {/  /\\  \\| || .-. \\   ",
			"`----' `-' `---'  `--'`-'  `-'`-'`-' `-'   ",
		}, "\n"),
		smallLogoStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")),
		framePadX:      2,
		framePadY:      1,
	}
	if key, err := lib.NewSfFolder().GetKey(); err == nil && key != "" {
		m.loggedIn = true
		m.apiKey = key
		m.step = mainStepMenu
	} else {
		m.step = mainStepLogin
		m.inpApi.Focus()
	}
	return m
}

func (m mainModel) Init() tea.Cmd { return tea.Batch(m.sp.Tick, m.filepicker.Init()) }

// Update 与 View 的详细逻辑保持与原实现一致，只保留高层，细节调用已拆分函数
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		innerW, innerH := m.computeInnerSizeFor(msg.Width, msg.Height)
		lw := innerW - 6
		if lw < 20 {
			lw = 20
		}
		m.menu.SetWidth(lw)
		m.typeList.SetWidth(lw)
		m.baseList.SetWidth(lw)
		m.coverMethodList.SetWidth(lw)
		m.introMethodList.SetWidth(lw)
		m.moreList.SetWidth(lw)
		m.inpApi.Width = lw
		m.inpName.Width = lw
		m.inpVersion.Width = lw
		m.inpPath.Width = lw
		m.inpCover.Width = lw
		m.inpExt.Width = lw
		m.taIntro.SetWidth(lw - 2)
		// 动态调整 textarea 高度
		// 为标题、字符计数、提示文字等预留约 6 行空间
		taHeight := innerH - 6
		if taHeight < 8 {
			taHeight = 8 // 最小高度 8 行
		}
		if taHeight > 40 {
			taHeight = 40 // 最大高度 40 行，避免过大
		}
		m.taIntro.SetHeight(taHeight)
		m.progress.Width = innerW - 10
		if m.progress.Width < 10 {
			m.progress.Width = 10
		}
		if len(m.verProgress) > 0 {
			for i := range m.verProgress {
				m.verProgress[i].Width = innerW - 10
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}
		if innerH > 15 {
			m.filepicker.SetHeight(innerH - 19)
		} else {
			m.filepicker.SetHeight(5)
		}
		m.resizeModelTable(innerW)

		// 动态调整模型详情 viewport 大小
		detailViewportHeight := innerH - 12
		if detailViewportHeight < 5 {
			detailViewportHeight = 5
		}
		detailViewportWidth := innerW - 4
		if detailViewportWidth < 40 {
			detailViewportWidth = 40
		}

		// 检查尺寸是否变化
		needRerender := false
		if m.detailViewport.Width != detailViewportWidth || m.detailViewport.Height != detailViewportHeight {
			m.detailViewport.Width = detailViewportWidth
			m.detailViewport.Height = detailViewportHeight
			needRerender = true
		}

		// 如果正在查看详情且尺寸变化了，需要重新渲染内容
		if needRerender && m.viewingModelDetail && m.modelDetail != nil && m.detailContent != "" {
			// 重新生成 Markdown
			markdown := buildModelDetailMarkdown(m.modelDetail)

			// 使用新的宽度重新渲染
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(detailViewportWidth),
			)

			if err == nil {
				if rendered, err := renderer.Render(markdown); err == nil {
					m.detailContent = rendered
					// 保持当前滚动位置（如果可能）
					yOffset := m.detailViewport.YOffset
					m.detailViewport.SetContent(m.detailContent)
					m.detailViewport.YOffset = yOffset
					// 确保不超出范围
					if m.detailViewport.YOffset > m.detailViewport.TotalLineCount()-m.detailViewport.Height {
						m.detailViewport.YOffset = m.detailViewport.TotalLineCount() - m.detailViewport.Height
						if m.detailViewport.YOffset < 0 {
							m.detailViewport.YOffset = 0
						}
					}
				}
			}
		}

		return m, nil
	case tea.KeyMsg:
		if m.err != nil {
			if m.step != mainStepOutput {
				m.err = nil
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c":
			// 如果正在上传，保持取消上传逻辑
			if m.running && m.currentAction == actionUpload && m.cancelFn != nil {
				// 设置取消标志，等待后台任务完成
				if !m.canceling {
					m.cancelFn()
					m.canceling = true
				}
				return m, nil
			}
			// 如果已经在确认退出状态，直接退出
			if m.confirmingExit {
				return m, tea.Quit
			}
			// 否则进入退出确认状态
			m.confirmingExit = true
			return m, nil
		case "enter":
			// 如果在退出确认界面，Enter 确认退出
			if m.confirmingExit {
				return m, tea.Quit
			}
			// 登录/菜单/输出页行为保持
			// 其余交给 updateActionInputs
			switch m.step {
			case mainStepLogin:
				api := m.inpApi.Value()
				if api == "" {
					m.err = fmt.Errorf("API Key 不能为空")
					return m, nil
				}
				m.running = true
				return m, loginCmd(api)
			case mainStepMenu:
				if it, ok := m.menu.SelectedItem().(menuEntry); ok {
					m.currentAction = it.key
					switch it.key {
					case actionExit:
						return m, tea.Quit
					case actionLogout:
						m.running = true
						return m, runLogout()
					case actionWhoami:
						m.running = true
						return m, runWhoami(m.apiKey)
					case actionUpload:
						m.step = mainStepAction
						m.act = actionInputs{}
						m.upStep = stepType
						// 如果还没有加载基础模型类型，则开始加载
						if len(m.baseModelTypes) == 0 && !m.loadingBaseModelTypes {
							m.loadingBaseModelTypes = true
							return m, loadBaseModelTypes(m.apiKey)
						}
						return m, nil
					case actionLsModel:
						m.step = mainStepAction
						m.act = actionInputs{}
						m.loadingModelList = true
						return m, loadModelList(m.apiKey)
					}
				}
			case mainStepAction:
				return m, m.updateActionInputs(msg)
			case mainStepOutput:
				m.step = mainStepMenu
				m.output = ""
				m.err = nil
				m.resetUploadState()
				return m, nil
			}
		case "esc":
			// 如果在退出确认界面，Esc 取消退出
			if m.confirmingExit {
				m.confirmingExit = false
				return m, nil
			}
			switch m.step {
			case mainStepLogin:
				m.step = mainStepHome
				return m, nil
			case mainStepAction:
				return m, m.updateActionInputs(msg)
			case mainStepOutput:
				m.step = mainStepMenu
				m.output = ""
				m.err = nil
				m.resetUploadState()
				return m, nil
			}
		default:
			switch m.step {
			case mainStepLogin:
				var cmd tea.Cmd
				m.inpApi, cmd = m.inpApi.Update(msg)
				return m, cmd
			case mainStepMenu:
				var cmd tea.Cmd
				m.menu, cmd = m.menu.Update(msg)
				return m, cmd
			case mainStepAction:
				return m, m.updateActionInputs(msg)
			}
		}
	case loginDoneMsg:
		m.running = false
		if msg.err != nil || !msg.ok {
			if msg.err != nil {
				m.err = msg.err
			} else {
				m.err = fmt.Errorf("登录失败")
			}
			return m, nil
		}
		m.loggedIn = true
		m.apiKey = m.inpApi.Value()
		m.step = mainStepMenu
		return m, nil
	case modelListLoadedMsg:
		m.loadingModelList = false
		if msg.err != nil {
			m.err = withStep("加载模型列表", msg.err)
			m.step = mainStepOutput
			m.output = fmt.Sprintf("加载模型列表失败: %v", msg.err)
			return m, nil
		}
		m.modelList = msg.models
		m.modelListTotal = msg.total
		m.rebuildModelTableRows()
		return m, nil
	case modelDetailLoadedMsg:
		m.loadingModelDetail = false
		if msg.err != nil {
			m.err = msg.err
			m.viewingModelDetail = false
			return m, nil
		}
		m.modelDetail = msg.detail

		// 生成 Markdown 内容并使用 glamour 渲染
		markdown := buildModelDetailMarkdown(msg.detail)

		// 使用 glamour 渲染 Markdown，使用深色主题并设置合适的宽度
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.detailViewport.Width),
		)

		if err != nil {
			// 如果 glamour 初始化失败，降级到纯文本显示
			m.detailContent = markdown
		} else {
			rendered, err := renderer.Render(markdown)
			if err != nil {
				// 如果渲染失败，降级到纯文本显示
				m.detailContent = markdown
			} else {
				m.detailContent = rendered
			}
		}

		// 设置到 viewport 并重置滚动位置
		m.detailViewport.SetContent(m.detailContent)
		m.detailViewport.GotoTop()

		return m, nil
	case deleteModelDoneMsg:
		m.running = false
		m.confirmingDelete = false
		m.deleteTargetId = 0
		m.deleteTargetName = ""
		m.viewingModelDetail = false
		m.modelDetail = nil
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.loadingModelList = true
		return m, loadModelList(m.apiKey)
	case actionDoneMsg:
		m.running = false
		m.canceling = false

		// 特殊处理 logout：清除登录状态并返回到登录页面
		if m.currentAction == actionLogout {
			if msg.err != nil {
				// 即使登出失败也显示错误并返回登录页面
				m.err = msg.err
			}
			// 清除登录状态
			m.loggedIn = false
			m.apiKey = ""
			m.step = mainStepLogin
			m.inpApi.SetValue("")
			m.inpApi.Focus() // 设置焦点以便用户可以输入新的 API Key
			return m, nil
		}

		m.output = msg.out
		if msg.err != nil {
			m.err = msg.err
		}
		m.step = mainStepOutput
		m.resetUploadState()
		return m, nil
	case uploadStartMsg:
		m.uploadCh = msg.ch
		m.cancelFn = msg.cancel
		if len(m.act.versions) > 0 {
			m.verProgress = make([]progress.Model, len(m.act.versions))
			m.verConsumed = make([]int64, len(m.act.versions))
			m.verTotal = make([]int64, len(m.act.versions))
			m.verLastTime = make([]time.Time, len(m.act.versions))
			m.verLastBytes = make([]int64, len(m.act.versions))
			m.verSpeed = make([]int64, len(m.act.versions))
			for i := range m.verProgress {
				m.verProgress[i] = progress.New(progress.WithDefaultGradient())
				m.verProgress[i].Width = m.width - 10
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}
		return m, waitForUploadEvent(m.uploadCh)
	case uploadProgMsg:
		m.uploadProg = msg
		now := time.Now()
		var cmds []tea.Cmd

		if msg.total > 0 {
			if msg.verIdx >= 0 && msg.verIdx < len(m.verProgress) {
				// 多版本上传速率计算
				if !m.verLastTime[msg.verIdx].IsZero() {
					duration := now.Sub(m.verLastTime[msg.verIdx]).Seconds()
					if duration > 0 {
						bytesDiff := msg.consumed - m.verLastBytes[msg.verIdx]
						m.verSpeed[msg.verIdx] = int64(float64(bytesDiff) / duration)
					}
				}
				m.verLastTime[msg.verIdx] = now
				m.verLastBytes[msg.verIdx] = msg.consumed
				m.verConsumed[msg.verIdx] = msg.consumed
				m.verTotal[msg.verIdx] = msg.total
				cmds = append(cmds, m.verProgress[msg.verIdx].SetPercent(float64(msg.consumed)/float64(msg.total)))
			} else {
				// 单版本上传速率计算
				if !m.lastUploadTime.IsZero() {
					duration := now.Sub(m.lastUploadTime).Seconds()
					if duration > 0 {
						bytesDiff := msg.consumed - m.lastUploadBytes
						m.currentUploadSpeed = int64(float64(bytesDiff) / duration)
					}
				}
				m.lastUploadTime = now
				m.lastUploadBytes = msg.consumed
				cmds = append(cmds, m.progress.SetPercent(float64(msg.consumed)/float64(msg.total)))
			}
		}
		cmds = append(cmds, waitForUploadEvent(m.uploadCh))
		return m, tea.Batch(cmds...)
	case coverStatusMsg:
		m.coverStatus = msg.message
		m.coverStatusWarning = (msg.status == "fallback")
		return m, waitForUploadEvent(m.uploadCh)
	case clearFilePickerErrorMsg:
		m.act.filePickerErr = nil
		return m, nil
	case baseModelTypesLoadedMsg:
		m.loadingBaseModelTypes = false
		if msg.err != nil {
			// 如果加载失败，使用本地硬编码的列表作为后备
			bases := make([]string, 0, len(meta.SupportedBaseModels))
			for k := range meta.SupportedBaseModels {
				bases = append(bases, k)
			}
			sort.Strings(bases)
			bItems := []list.Item{}
			for _, b := range bases {
				bItems = append(bItems, listItem{title: b})
			}
			m.baseList.SetItems(bItems)
			return m, nil
		}
		// 加载成功，更新基础模型类型列表
		m.baseModelTypes = msg.items
		bItems := []list.Item{}
		for _, item := range msg.items {
			bItems = append(bItems, listItem{title: item.Value})
		}
		m.baseList.SetItems(bItems)
		return m, nil
	case checkModelExistsDoneMsg:
		m.running = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.exists {
			// 模型名重复，显示错误并停留在 stepName
			m.err = fmt.Errorf("模型名 '%s' 已存在，请换一个名字", m.act.u.name)
			m.inpName.SetValue("")
			m.act.u.name = ""
			return m, m.inpName.Focus()
		}
		// 模型名不重复，进入下一步
		m.upStep = stepVersion
		m.inpVersion.SetValue("v1.0")
		return m, m.inpVersion.Focus()
	default:
		var bat []tea.Cmd
		var cmd1 tea.Cmd
		m.sp, cmd1 = m.sp.Update(msg)
		if cmd1 != nil {
			bat = append(bat, cmd1)
		}
		if pModel, pCmd := m.progress.Update(msg); true {
			if pm, ok := pModel.(progress.Model); ok {
				m.progress = pm
			}
			if pCmd != nil {
				bat = append(bat, pCmd)
			}
		}
		for i := range m.verProgress {
			if pModel, pCmd := m.verProgress[i].Update(msg); true {
				if pm, ok := pModel.(progress.Model); ok {
					m.verProgress[i] = pm
				}
				if pCmd != nil {
					bat = append(bat, pCmd)
				}
			}
		}
		if len(bat) > 0 {
			return m, tea.Batch(bat...)
		}
		if m.step == mainStepAction && m.currentAction == actionUpload && m.act.useFilePicker {
			var fpCmd tea.Cmd
			m.filepicker, fpCmd = m.filepicker.Update(msg)
			if fpCmd != nil {
				return m, fpCmd
			}
		}
	}
	if m.loadingModelList {
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m mainModel) View() string {
	innerW, innerH := m.innerSize()
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 5 {
		innerH = 5
	}
	panelW := innerW - 4
	if panelW < 40 {
		panelW = 40
	}
	panel := m.panelStyle.Width(panelW)
	header := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, m.renderGradientLogo(m.smallLogo))

	// 退出确认界面优先级最高
	if m.confirmingExit {
		var b strings.Builder
		b.WriteString(m.titleStyle.Render("确认退出"))
		b.WriteString("\n\n")
		b.WriteString("确定要退出 BizyAir CLI 吗？\n\n")
		b.WriteString(m.hintStyle.Render("确认退出：Enter，取消：Esc，再次按 Ctrl+C 也可退出"))
		return m.renderFrame(header + "\n" + panel.Render(b.String()))
	}

	if m.err != nil && m.step != mainStepOutput {
		defer func() { m.err = nil }()
		var body strings.Builder
		if step := errStep(m.err); step != "" {
			body.WriteString(fmt.Sprintf("步骤: %s\n", step))
		}
		body.WriteString(fmt.Sprintf("错误: %v", m.err))
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("错误")+"\n"+body.String()+"\n\n"+m.hintStyle.Render("按任意键返回继续…")))
	}
	if m.running {
		if m.currentAction == actionUpload && m.step == mainStepAction {
			// 如果是在 stepName 步骤，显示校验模型名的提示
			if m.upStep == stepName {
				spin := m.sp.View()
				return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("上传 · Step 2/9 · 模型名称")+"\n\n"+m.inpName.View()+"\n\n"+spin+" 正在校验模型名是否重复…"))
			}
			return m.renderFrame(header + "\n" + panel.Render(m.renderUploadRunningView()))
		}
		spin := m.sp.View()
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…"))
	}
	switch m.step {
	case mainStepLogin:
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("登录 · 请输入 API Key")+"\n\n"+m.inpApi.View()+"\n"+m.hintStyle.Render("确认：Enter，返回：Esc，退出：Ctrl+C")))
	case mainStepMenu:
		logoStr := m.renderGradientLogo(m.logo)
		menuTop := strings.Count(logoStr, "\n") + 2
		h := innerH - menuTop - 6
		if h < 6 {
			h = 6
		}
		m.menu.SetHeight(h)
		return m.renderFrame(logoStr + "\n\n" + m.titleStyle.Render("功能选择") + "\n\n" + m.menu.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：Ctrl+C"))
	case mainStepAction:
		return m.renderFrame(header + "\n" + panel.Render(m.renderActionView()))
	case mainStepOutput:
		if m.err != nil {
			body := m.output
			if strings.TrimSpace(body) != "" {
				body += "\n"
			}
			if step := errStep(m.err); step != "" {
				body += fmt.Sprintf("步骤: %s\n", step)
			}
			body += fmt.Sprintf("错误: %v", m.err)
			return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("执行完成（含错误）")+"\n\n"+body+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单")))
		}
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("执行完成")+"\n\n"+m.output+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单")))
	default:
		if m.running {
			spin := m.sp.View()
			return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…"))
		}
		return m.renderFrame(header)
	}
}

// renderGradientLogo 为给定的logo文本应用渐变颜色（从紫色到粉色）
func (m mainModel) renderGradientLogo(logoText string) string {
	var logoB strings.Builder
	lines := strings.Split(logoText, "\n")
	n := len(lines)
	sr, sg, sb := hexToRGB("#8B5CF6") // 起始颜色：紫色
	er, eg, eb := hexToRGB("#EC4899") // 结束颜色：粉色
	for i, line := range lines {
		var t float64
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		r := lerpInt(sr, er, t)
		g := lerpInt(sg, eg, t)
		b := lerpInt(sb, eb, t)
		color := rgbToHex(r, g, b)
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
		logoB.WriteString(style.Render(line))
		logoB.WriteString("\n")
	}
	return strings.TrimRight(logoB.String(), "\n")
}

// 入口
func MainTUI(c *cli.Context) error {
	p := tea.NewProgram(newMainModel(), tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return err
	}
	if _, ok := model.(mainModel); ok {
		return nil
	}
	return nil
}
