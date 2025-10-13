package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	menu       list.Model
	inpApi     textinput.Model
	typeList   list.Model
	baseList   list.Model
	moreList   list.Model
	inpName    textinput.Model
	inpPath    textinput.Model
	inpCover   textinput.Model
	inpExt     textinput.Model
	inpVersion textinput.Model
	inpIntro   textinput.Model

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

	confirmingDelete bool
	deleteTargetId   int64
	deleteTargetName string

	act    actionInputs
	upStep uploadStep

	running     bool
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

	bases := make([]string, 0, len(meta.SupportedBaseModels))
	for k := range meta.SupportedBaseModels {
		bases = append(bases, k)
	}
	sort.Strings(bases)
	bItems := []list.Item{}
	for _, b := range bases {
		bItems = append(bItems, listItem{title: b})
	}
	bl := list.New(bItems, d, 30, 12)
	bl.Title = "选择 Base Model（必选）"

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
	inIntro := textinput.New()
	inIntro.Placeholder = "可选，输入模型介绍（回车跳过）"
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
	fp.ShowHidden = true
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.Height = 10
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

	m := mainModel{
		step:       mainStepHome,
		menu:       menuList,
		inpApi:     inApi,
		typeList:   tp,
		baseList:   bl,
		moreList:   ml,
		inpName:    inName,
		inpPath:    inPath,
		inpCover:   inCover,
		inpExt:     inExt,
		inpVersion: inVer,
		inpIntro:   inIntro,
		filepicker: fp,
		modelTable: modelTable,
		titleStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7")),
		hintStyle:  lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		panelStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2),
		btnStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#04B575")).Padding(0, 1).Bold(true),
		sp:         sp,
		progress:   pr,
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
		m.moreList.SetWidth(lw)
		m.inpApi.Width = lw
		m.inpName.Width = lw
		m.inpVersion.Width = lw
		m.inpPath.Width = lw
		m.inpCover.Width = lw
		m.inpExt.Width = lw
		m.inpIntro.Width = lw
		m.progress.Width = innerW - 6
		if m.progress.Width < 10 {
			m.progress.Width = 10
		}
		if len(m.verProgress) > 0 {
			for i := range m.verProgress {
				m.verProgress[i].Width = innerW - 6
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}
		if innerH > 15 {
			m.filepicker.SetHeight(innerH - 15)
		} else {
			m.filepicker.SetHeight(5)
		}
		m.resizeModelTable(innerW)
		return m, nil
	case tea.KeyMsg:
		if m.err != nil {
			if m.step != mainStepOutput {
				m.err = nil
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.running && m.currentAction == actionUpload && m.cancelFn != nil {
				m.cancelFn()
				m.uploadCh = nil
				m.running = false
				m.resetUploadState()
				return m, nil
			}
			return m, tea.Quit
		case "enter":
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
			for i := range m.verProgress {
				m.verProgress[i] = progress.New(progress.WithDefaultGradient())
				m.verProgress[i].Width = m.width - 6
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}
		return m, waitForUploadEvent(m.uploadCh)
	case uploadProgMsg:
		m.uploadProg = msg
		var cmds []tea.Cmd
		if msg.total > 0 {
			if msg.verIdx >= 0 && msg.verIdx < len(m.verProgress) {
				m.verConsumed[msg.verIdx] = msg.consumed
				m.verTotal[msg.verIdx] = msg.total
				cmds = append(cmds, m.verProgress[msg.verIdx].SetPercent(float64(msg.consumed)/float64(msg.total)))
			} else {
				cmds = append(cmds, m.progress.SetPercent(float64(msg.consumed)/float64(msg.total)))
			}
		}
		cmds = append(cmds, waitForUploadEvent(m.uploadCh))
		return m, tea.Batch(cmds...)
	case clearFilePickerErrorMsg:
		m.act.filePickerErr = nil
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
	header := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, m.smallLogoStyle.Render(m.smallLogo))

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
				return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("上传 · Step 2/8 · 模型名称")+"\n\n"+m.inpName.View()+"\n\n"+spin+" 正在校验模型名是否重复…"))
			}
			return m.renderFrame(header + "\n" + panel.Render(m.renderUploadRunningView()))
		}
		spin := m.sp.View()
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…"))
	}
	switch m.step {
	case mainStepLogin:
		return m.renderFrame(header + "\n" + panel.Render(m.titleStyle.Render("登录 · 请输入 API Key")+"\n\n"+m.inpApi.View()+"\n"+m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")))
	case mainStepMenu:
		var logoB strings.Builder
		lines := strings.Split(m.logo, "\n")
		n := len(lines)
		sr, sg, sb := hexToRGB("#8B5CF6")
		er, eg, eb := hexToRGB("#EC4899")
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
		logoStr := strings.TrimRight(logoB.String(), "\n")
		menuTop := strings.Count(logoStr, "\n") + 2
		h := innerH - menuTop - 6
		if h < 6 {
			h = 6
		}
		m.menu.SetHeight(h)
		return m.renderFrame(logoStr + "\n\n" + m.titleStyle.Render("功能选择") + "\n\n" + m.menu.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q"))
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
