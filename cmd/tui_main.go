package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// 主 TUI：统一入口（Logo -> 登录/菜单 -> 功能执行）

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
	actionLsFiles actionKind = "ls_files"
	actionRmModel actionKind = "rm_model"
	actionWhoami  actionKind = "whoami"
	actionLogout  actionKind = "logout"
	actionExit    actionKind = "exit"
)

// 上传步骤状态机
type uploadStep int

const (
	stepName uploadStep = iota
	stepType
	stepVersion
	stepBase
	stepCover
	stepIntro
	stepPath
	stepAskMore
	stepConfirm
)

// 登录结果消息
type loginDoneMsg struct {
	ok  bool
	err error
}

// 统一命令执行结果
type actionDoneMsg struct {
	out string
	err error
}

// 上传开始（携带进度通道与取消函数）
type uploadStartMsg struct {
	ch     <-chan tea.Msg
	cancel context.CancelFunc
}

// 上传进度
type uploadProgMsg struct {
	fileIndex string
	fileName  string
	consumed  int64
	total     int64
	verIdx    int
}

// 取消上传
type uploadCancelMsg struct{}

// 清除filepicker错误的消息
type clearFilePickerErrorMsg struct{}

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

// 针对不同动作的输入组件（尽量精简，必要时逐步扩展）
type actionInputs struct {
	// 通用
	confirming bool

	// Upload
	u uploadInputs
	// 多版本
	versions []versionItem
	cur      versionItem

	// List Models
	lsPublic bool

	// List Files
	lfPublic  bool
	lfTree    bool
	lfExt     string
	lfExtDone bool

	// Filepicker state
	useFilePicker bool
	filePickerErr error

	// 路径输入相关
	pathInputFocused bool
}

type mainModel struct {
	step          mainStep
	loggedIn      bool
	apiKey        string
	err           error
	currentAction actionKind

	// 尺寸
	width  int
	height int

	// 组件
	menu   list.Model
	inpApi textinput.Model
	// 上传子组件（选择/输入）
	typeList list.Model
	baseList list.Model
	moreList list.Model
	inpName  textinput.Model
	inpPath  textinput.Model
	inpCover textinput.Model
	// 其他输入
	inpExt textinput.Model
	// 新增输入
	inpVersion textinput.Model
	inpIntro   textinput.Model

	//文件选择器
	filepicker   filepicker.Model
	selectedFile string

	// 动作输入
	act    actionInputs
	upStep uploadStep

	// 运行态
	running  bool
	output   string
	sp       spinner.Model
	progress progress.Model
	// 多版本进度
	verProgress []progress.Model
	verConsumed []int64
	verTotal    []int64

	// 样式
	titleStyle lipgloss.Style
	hintStyle  lipgloss.Style
	panelStyle lipgloss.Style
	btnStyle   lipgloss.Style

	// 品牌
	logo           string
	logoStyle      lipgloss.Style
	smallLogoStyle lipgloss.Style

	// 上传进度
	uploadCh   <-chan tea.Msg
	uploadProg uploadProgMsg
	cancelFn   context.CancelFunc
}

func newMainModel() mainModel {
	// 菜单项
	mItems := []list.Item{
		menuEntry{listItem{title: "上传模型", desc: "交互式收集参数并上传"}, actionUpload},
		menuEntry{listItem{title: "列出模型", desc: "按类型浏览模型"}, actionLsModel},
		menuEntry{listItem{title: "查看模型文件", desc: "按类型与名称查看文件"}, actionLsFiles},
		menuEntry{listItem{title: "删除模型", desc: "按类型与名称删除模型"}, actionRmModel},
		menuEntry{listItem{title: "当前账户信息", desc: "显示 whoami"}, actionWhoami},
		menuEntry{listItem{title: "退出登录", desc: "清除本地 API Key"}, actionLogout},
		menuEntry{listItem{title: "退出程序", desc: "离开 BizyAir CLI"}, actionExit},
	}

	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#04B575")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)

	menuList := list.New(mItems, d, 30, len(mItems))
	menuList.Title = "请选择功能"
	menuList.SetShowStatusBar(false)
	menuList.SetShowPagination(false)

	// 上传/通用：类型列表
	tItems := make([]list.Item, 0, len(meta.ModelTypes))
	for _, t := range meta.ModelTypes {
		s := string(t)
		tItems = append(tItems, listItem{title: s})
	}
	tp := list.New(tItems, d, 30, len(tItems))
	tp.Title = "选择模型类型"
	tp.SetShowStatusBar(false)
	tp.SetShowPagination(false)

	// 上传：Base 列表
	bases := make([]string, 0, len(meta.SupportedBaseModels))
	for k := range meta.SupportedBaseModels {
		bases = append(bases, k)
	}
	sort.Strings(bases)
	bItems := []list.Item{listItem{title: "(跳过)", desc: "可不选择 Base Model"}}
	for _, b := range bases {
		bItems = append(bItems, listItem{title: b})
	}
	bl := list.New(bItems, d, 30, 12)
	bl.Title = "选择 Base Model（可选）"

	// 是否添加更多版本选择
	moreItems := []list.Item{listItem{title: "是，继续添加版本"}, listItem{title: "否，进入确认"}}
	ml := list.New(moreItems, d, 30, 4)
	ml.Title = "是否继续添加版本？请按方向键进行选择，Enter确认"

	// 输入框
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
	// 设置默认值为用户主目录
	if homeDir, err := os.UserHomeDir(); err == nil {
		inPath.SetValue(homeDir + "/")
	}

	inCover := textinput.New()
	inCover.Placeholder = "可选，多地址以 ; 分隔"

	inExt := textinput.New()
	inExt.Placeholder = "可选，文件扩展名（如 .safetensors）"

	//文件选择器 - 配置为支持模型文件格式
	fp := filepicker.New()
	// 允许所有文件类型，让用户可以浏览
	fp.AllowedTypes = []string{} // 空数组表示允许所有文件类型

	// 设置起始目录为用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	fp.CurrentDirectory = homeDir
	fp.ShowHidden = true // 显示隐藏文件，这样可以看到更多文件
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.Height = 10 // 设置一个合理的高度
	fp.AutoHeight = false

	// 自定义样式，提供更友好的空目录消息
	fp.Styles.EmptyDirectory = fp.Styles.EmptyDirectory.SetString("此目录为空。使用方向键导航到其他目录。")

	// spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	// progress bar
	pr := progress.New(progress.WithDefaultGradient())

	return mainModel{
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
		titleStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7")),
		hintStyle:  lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		panelStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2),
		btnStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#04B575")).Padding(0, 1).Bold(true),
		sp:         sp,
		progress:   pr,
		logo: strings.Join([]string{
			"    ,---,.                                     ,---,                         ",
			" ,'  .'  \\  ,--,                             '  .' \\        ,--,              ",
			",---.' .' |,--.'|          ,----,            /  ;    '.    ,--.'|    __  ,-. ",
			"|   |  |: ||  |,         .'   .`|           :  :       \\   |  |,   ,' ,'/ /| ",
			";   :  :  /`--'_      .'   .'  .'      .--, :  |   /\\   \\  `--'_   '  | |' | ",
			";   |    ; ,' ,'|   ,---, '   ./     /_ ./| |  :  ' ;.   : ,' ,'|  |  |   ,' ",
			"|   :     \\  | |   ;   | .'  /   , ' , ' : |  |  ;/  \\   \\  | |  '  :  /   ",
			"|   |   . ||  | :   `---' /  ;--,/___/ \\: | '  :  | \\  \\ ,'|  | :  |  | '    ",
			"'   :  '; |'  : |__   /  /  / .`| .  \\  ' | |  |  '  '--'  '  : |__;  : |    ",
			"|   |  | ; |  | '.'|./__;     .'   \\  ;   : |  :  :        |  | '.'|  , ;    ",
			"|   :   /  ;  :    ;;   |  .'       \\  \\  ; |  | ,'        ;  :    ;---'     ",
			"|   | ,'   |  ,   / `---'            :  \\  \\`--''          |  ,   /          ",
			"`----'      ---`-'                    \\  ' ;                ---`-'           ",
			"                                        `--`                                     ",
		}, "\n"),
		logoStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")),
		smallLogoStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")),
	}
}

// 清除filepicker错误的命令
func clearFilePickerErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearFilePickerErrorMsg{}
	})
}

// 等待上传事件（进度/完成）
func waitForUploadEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m mainModel) Init() tea.Cmd { return tea.Batch(m.sp.Tick, m.filepicker.Init()) }

// 从文件路径提取文件名（不含扩展名）作为模型名
func extractModelNameFromPath(filePath string) string {
	// 获取文件名（含扩展名）
	fileName := filepath.Base(filePath)
	// 去除扩展名
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// 清理文件名，只保留字母、数字、下划线和短横线
	var result strings.Builder
	for _, r := range nameWithoutExt {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else if r == ' ' || r == '.' {
			result.WriteRune('_')
		}
	}
	cleanName := result.String()
	// 确保不为空，如果为空则使用默认名称
	if cleanName == "" {
		cleanName = "model"
	}
	return cleanName
}

// 验证并设置路径的辅助方法
func (m *mainModel) validateAndSetPath(path string) error {
	// 验证文件扩展名
	supportedExts := []string{".safetensors", ".bin", ".ckpt", ".pt", ".pth", ".pkl", ".h5", ".onnx", ".tflite", ".pb", ".json", ".txt", ".md", ".yaml", ".yml"}
	isSupported := false
	for _, ext := range supportedExts {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("不支持的文件格式，支持的格式：%s", strings.Join(supportedExts, ", "))
	}

	if err := validatePath(path); err != nil {
		return err
	}

	m.act.cur.path = absPath(path)
	m.selectedFile = path
	m.act.filePickerErr = nil

	// 提取文件名作为name的默认值
	defaultName := extractModelNameFromPath(path)
	m.inpName.SetValue(defaultName)

	return nil
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		lw := msg.Width - 6
		if lw < 20 {
			lw = 20
		}
		m.menu.SetWidth(lw)
		m.typeList.SetWidth(lw)
		m.baseList.SetWidth(lw)
		m.inpApi.Width = lw
		m.inpName.Width = lw
		m.inpPath.Width = lw
		m.inpCover.Width = lw
		m.inpExt.Width = lw
		m.inpIntro.Width = lw

		// 进度条宽度自适应屏幕
		m.progress.Width = msg.Width - 6
		if m.progress.Width < 10 {
			m.progress.Width = 10
		}

		// 多版本进度条宽度自适应
		if len(m.verProgress) > 0 {
			for i := range m.verProgress {
				m.verProgress[i].Width = msg.Width - 6
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}

		// 设置filepicker的高度
		if m.height > 15 {
			m.filepicker.SetHeight(m.height - 15)
		} else {
			m.filepicker.SetHeight(5)
		}
		return m, nil
	case tea.KeyMsg:
		if m.err != nil {
			m.err = nil
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.running && m.currentAction == actionUpload && m.cancelFn != nil {
				m.cancelFn()
				// 停止等待更多上传事件
				m.uploadCh = nil
				m.running = false
				return m, nil
			}
			return m, tea.Quit
		case "enter":
			switch m.step {
			case mainStepHome:
				// 检测登录
				if key, err := lib.NewSfFolder().GetKey(); err == nil && key != "" {
					m.loggedIn = true
					m.apiKey = key
					m.step = mainStepMenu
					return m, nil
				}
				m.step = mainStepLogin
				return m, m.inpApi.Focus()
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
						m.upStep = stepName
						// 聚焦到名称输入
						return m, m.inpName.Focus()
					case actionLsModel:
						m.step = mainStepAction
						m.act = actionInputs{}
						return m, nil
					case actionLsFiles:
						m.step = mainStepAction
						m.act = actionInputs{}
						return m, nil
					case actionRmModel:
						m.step = mainStepAction
						m.act = actionInputs{}
						return m, nil
					}
				}
			case mainStepAction:
				return m, m.updateActionInputs(msg)
			case mainStepOutput:
				// 回菜单
				m.step = mainStepMenu
				m.output = ""
				m.running = false
				m.act = actionInputs{}
				m.selectedFile = ""
				m.filepicker.Path = ""
				return m, nil
			}
		case "esc":
			switch m.step {
			case mainStepLogin:
				m.step = mainStepHome
				return m, nil
			case mainStepAction:
				// 回退由 updateActionInputs 处理（根据当前动作与子步骤）
				return m, m.updateActionInputs(msg)
			case mainStepOutput:
				m.step = mainStepMenu
				m.output = ""
				m.running = false
				m.act = actionInputs{}
				m.selectedFile = ""
				m.filepicker.Path = ""
				return m, nil
			}
		default:
			// 非 enter 的任意键在登录错误态下清错已在上面处理
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
	case actionDoneMsg:
		m.running = false
		m.output = msg.out
		if msg.err != nil {
			m.err = msg.err
		}
		m.step = mainStepOutput
		m.uploadCh = nil
		m.uploadProg = uploadProgMsg{}
		return m, nil
	case uploadStartMsg:
		m.uploadCh = msg.ch
		m.cancelFn = msg.cancel
		// 初始化多版本进度条集合
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
		// 更新单/多版本进度
		var cmds []tea.Cmd
		if msg.total > 0 {
			percent := float64(msg.consumed) / float64(msg.total)
			if msg.verIdx >= 0 && msg.verIdx < len(m.verProgress) {
				m.verConsumed[msg.verIdx] = msg.consumed
				m.verTotal[msg.verIdx] = msg.total
				cmds = append(cmds, m.verProgress[msg.verIdx].SetPercent(percent))
			} else {
				cmds = append(cmds, m.progress.SetPercent(percent))
			}
		}
		cmds = append(cmds, waitForUploadEvent(m.uploadCh))
		return m, tea.Batch(cmds...)
	case clearFilePickerErrorMsg:
		m.act.filePickerErr = nil
		return m, nil
	default:
		// 转给 spinner 和所有 progress（组合返回动画命令）
		var bat []tea.Cmd
		var cmd1 tea.Cmd
		m.sp, cmd1 = m.sp.Update(msg)
		if cmd1 != nil {
			bat = append(bat, cmd1)
		}
		// 单进度
		if pModel, pCmd := m.progress.Update(msg); true {
			if pm, ok := pModel.(progress.Model); ok {
				m.progress = pm
			}
			if pCmd != nil {
				bat = append(bat, pCmd)
			}
		}
		// 多版本进度
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

		// 更新filepicker
		if m.step == mainStepAction && m.currentAction == actionUpload && m.act.useFilePicker {
			var fpCmd tea.Cmd
			m.filepicker, fpCmd = m.filepicker.Update(msg)
			if fpCmd != nil {
				return m, fpCmd
			}
		}
	}

	return m, nil
}

func (m mainModel) View() string {
	panelW := m.width - 4
	if panelW < 40 {
		panelW = 40
	}
	panel := m.panelStyle.Width(panelW)

	if m.step == mainStepHome {
		var homeBuilder strings.Builder
		lines := strings.Split(m.logo, "\n")
		n := len(lines)
		sr, sg, sb := hexToRGB("#8B5CF6") // 紫色
		er, eg, eb := hexToRGB("#EC4899") // 粉色
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
			homeBuilder.WriteString(style.Render(line))
			homeBuilder.WriteString("\n")
		}
		// 欢迎文案使用粉色收尾
		homeBuilder.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")).Render("欢迎使用 BizyAir CLI 工具"))
		return strings.TrimRight(homeBuilder.String(), "\n") + "\n" + panel.Render(m.hintStyle.Render("按 Enter 进入 ››"))
	}

	header := lipgloss.PlaceHorizontal(m.width, lipgloss.Left, m.smallLogoStyle.Render("BizyAir CLI"))

	if m.err != nil && m.step != mainStepOutput {
		defer func() { m.err = nil }()
		return header + "\n" + panel.Render(m.titleStyle.Render("错误")+"\n"+fmt.Sprintf("%v", m.err)+"\n\n"+m.hintStyle.Render("按任意键返回继续…"))
	}

	// 运行中：优先在"上传确认页"内联展示进度条，其它情况走通用覆盖
	if m.running {
		if m.currentAction == actionUpload && m.step == mainStepAction {
			// 构造确认摘要
			var summaryBuilder strings.Builder
			summaryBuilder.WriteString(fmt.Sprintf("- type: %s\n- name: %s\n", dash(m.act.u.typ), dash(m.act.u.name)))
			for i, v := range m.act.versions {
				summaryBuilder.WriteString(fmt.Sprintf("  [%d] version=%s base=%s cover=%s path=%s intro=%s\n", i+1, dash(v.version), dash(v.base), dash(v.cover), dash(v.path), dash(v.intro)))
			}
			summary := summaryBuilder.String()
			// 构造进度显示
			var fileLine string
			var progLine string
			if m.uploadProg.total > 0 {
				percent := float64(m.uploadProg.consumed) / float64(m.uploadProg.total)
				fileLine = fmt.Sprintf("(%s) %s", m.uploadProg.fileIndex, m.uploadProg.fileName)
				progLine = fmt.Sprintf("%s %.1f%% (%s/%s)", m.progress.View(), percent*100, formatBytes(m.uploadProg.consumed), formatBytes(m.uploadProg.total))
			} else {
				fileLine = "准备上传…"
				progLine = m.progress.View()
			}

			content := m.titleStyle.Render("上传中 · 请稍候") + "\n\n" + summary + "\n\n" + fileLine + "\n" + progLine
			return header + "\n" + panel.Render(content)
		}
		spin := m.sp.View()
		return header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…")
	}

	switch m.step {
	case mainStepLogin:
		return header + "\n" + panel.Render(m.titleStyle.Render("登录 · 请输入 API Key")+"\n\n"+m.inpApi.View()+"\n"+m.hintStyle.Render("确认：Enter，返回：Esc，退出：q"))
	case mainStepMenu:
		if m.height > 0 {
			h := m.height - 10
			if h < 6 {
				h = 6
			}
			m.menu.SetHeight(h)
		}
		return header + "\n" + panel.Render(m.titleStyle.Render("功能选择")+"\n\n"+m.menu.View()+"\n"+m.hintStyle.Render("确认：Enter，返回：Esc，退出：q"))
	case mainStepAction:
		return header + "\n" + panel.Render(m.renderActionView())
	case mainStepOutput:
		if m.err != nil {
			return header + "\n" + panel.Render(m.titleStyle.Render("执行完成（含错误）")+"\n\n"+m.output+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单"))
		}
		return header + "\n" + panel.Render(m.titleStyle.Render("执行完成")+"\n\n"+m.output+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单"))
	default:
		if m.running {
			spin := m.sp.View()
			return header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…")
		}
		return header
	}
}

// 工具：颜色插值与转换（渐变）
func rgbToHex(r, g, b int) string {
	return "#" + toHex(r) + toHex(g) + toHex(b)
}

func toHex(v int) string {
	h := strconv.FormatInt(int64(v), 16)
	if len(h) == 1 {
		h = "0" + h
	}
	return strings.ToUpper(h)
}

func hexToRGB(hex string) (int, int, int) {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return 255, 255, 255
	}
	r, _ := strconv.ParseInt(s[0:2], 16, 0)
	g, _ := strconv.ParseInt(s[2:4], 16, 0)
	b, _ := strconv.ParseInt(s[4:6], 16, 0)
	return int(r), int(g), int(b)
}

func lerpInt(a, b int, t float64) int {
	x := float64(a) + (float64(b)-float64(a))*t
	if x < 0 {
		x = 0
	}
	if x > 255 {
		x = 255
	}
	return int(math.Round(x))
}

// 根据当前动作处理输入与触发命令
func (m *mainModel) updateActionInputs(msg tea.Msg) tea.Cmd {
	// 目前仅实现上传流程，其它功能先占位后续扩展
	switch m.currentAction {
	case actionUpload:
		var cmd tea.Cmd
		switch m.upStep {
		case stepName:
			m.inpName, cmd = m.inpName.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					name := strings.TrimSpace(m.inpName.Value())
					if err := validateName(name); err != nil {
						m.err = err
						return nil
					}
					m.act.u.name = name
					m.upStep = stepType
					return nil
				case "esc":
					m.step = mainStepMenu
					m.act = actionInputs{}
					return nil
				}
			}
			return cmd
		case stepType:
			m.typeList, cmd = m.typeList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					if it, ok := m.typeList.SelectedItem().(listItem); ok {
						m.act.u.typ = it.title
						// 版本默认 v1.0
						m.inpVersion.SetValue("v1.0")
						m.upStep = stepVersion
						return nil
					}
				case "esc":
					m.upStep = stepName
					return m.inpName.Focus()
				}
			}
			return cmd
		case stepVersion:
			m.inpVersion, cmd = m.inpVersion.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					v := strings.TrimSpace(m.inpVersion.Value())
					if v == "" {
						v = "v1.0"
						m.inpVersion.SetValue(v)
					}
					m.act.cur.version = v
					m.upStep = stepBase
					return nil
				case "esc":
					m.upStep = stepType
					return nil
				}
			}
			return cmd
		case stepBase:
			m.baseList, cmd = m.baseList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					if it, ok := m.baseList.SelectedItem().(listItem); ok {
						if it.title != "(跳过)" {
							if !meta.SupportedBaseModels[it.title] {
								m.err = fmt.Errorf("不支持的 Base Model: %s", it.title)
								return nil
							}
							m.act.cur.base = it.title
						} else {
							m.act.cur.base = ""
						}
						m.upStep = stepCover
						return m.inpCover.Focus()
					}
				case "esc":
					m.upStep = stepVersion
					return nil
				}
			}
			return cmd
		case stepCover:
			var inputCmd tea.Cmd
			m.inpCover, inputCmd = m.inpCover.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					m.act.cur.cover = strings.TrimSpace(m.inpCover.Value())
					m.upStep = stepIntro
					return m.inpIntro.Focus()
				case "esc":
					m.upStep = stepBase
					return nil
				}
			}
			return inputCmd
		case stepIntro:
			m.inpIntro, cmd = m.inpIntro.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					// 允许为空
					m.act.cur.intro = strings.TrimSpace(m.inpIntro.Value())
					// 进入文件选择
					m.act.useFilePicker = true
					m.act.pathInputFocused = false
					m.inpPath.SetValue(m.filepicker.CurrentDirectory + "/")
					m.upStep = stepPath
					return m.filepicker.Init()
				case "esc":
					m.upStep = stepCover
					return m.inpCover.Focus()
				}
			}
			return cmd
		case stepPath:
			// 混合模式：路径输入框 + 文件选择器，类似官方示例
			var pathCmd, fpCmd tea.Cmd

			// 处理按键事件
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "esc":
					// 返回上一步（intro）
					m.act.useFilePicker = false
					m.act.pathInputFocused = false
					m.act.filePickerErr = nil
					m.filepicker.Path = ""
					m.upStep = stepIntro
					return m.inpIntro.Focus()
				case "ctrl+r":
					// 强制刷新：同步路径输入框到文件选择器
					path := strings.TrimSpace(m.inpPath.Value())
					if path != "" {
						if info, err := os.Stat(path); err == nil && info.IsDir() {
							m.filepicker.CurrentDirectory = path
							return m.filepicker.Init()
						} else {
							m.act.filePickerErr = fmt.Errorf("路径无效或不是目录: %s", path)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
					}
					// 记录路径并继续
					m.act.cur.path = absPath(path)
					// 完成本版本，进入是否添加更多版本
					m.upStep = stepAskMore
					return nil
				case "tab":
					// Tab键在路径输入框和文件选择器之间切换焦点
					if m.act.pathInputFocused {
						// 从路径输入框切换到文件选择器时，检查路径是否是目录
						path := strings.TrimSpace(m.inpPath.Value())
						if path != "" {
							if info, err := os.Stat(path); err == nil && info.IsDir() {
								// 如果路径是有效目录，更新filepicker
								m.filepicker.CurrentDirectory = path
								m.act.pathInputFocused = false
								m.inpPath.Blur()
								return m.filepicker.Init()
							}
						}
						// 如果路径无效，仍然切换焦点但不更新目录
						m.act.pathInputFocused = false
						m.inpPath.Blur()
						return nil
					} else {
						// 切换到路径输入框
						m.act.pathInputFocused = true
						return m.inpPath.Focus()
					}
				case "enter":
					if m.act.pathInputFocused && m.inpPath.Value() != "" {
						// 从输入框确认路径
						path := strings.TrimSpace(m.inpPath.Value())

						// 检查路径是否存在
						info, err := os.Stat(path)
						if err != nil {
							m.act.filePickerErr = fmt.Errorf("路径不存在: %s", path)
							return clearFilePickerErrorAfter(3 * time.Second)
						}

						if info.IsDir() {
							// 如果是目录，更新filepicker到该目录
							m.filepicker.CurrentDirectory = path
							return m.filepicker.Init()
						} else {
							// 如果是文件，直接选择该文件
							if err := m.validateAndSetPath(path); err != nil {
								m.act.filePickerErr = err
								return clearFilePickerErrorAfter(3 * time.Second)
							}
							return nil
						}
					}
				}
			}

			// 更新路径输入框
			if m.act.pathInputFocused {
				m.inpPath, pathCmd = m.inpPath.Update(msg)
			}

			// 更新文件选择器 - 只有在文件选择器有焦点时才更新
			if !m.act.pathInputFocused {
				oldDir := m.filepicker.CurrentDirectory
				m.filepicker, fpCmd = m.filepicker.Update(msg)

				// 如果目录改变了，同步更新路径输入框
				if m.filepicker.CurrentDirectory != oldDir {
					m.inpPath.SetValue(m.filepicker.CurrentDirectory + "/")
				}
			}

			// 检查文件选择 - 只有在文件选择器有焦点时才检查
			if !m.act.pathInputFocused {
				// 检查真正的文件选择（非文件夹）
				if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
					// 使用stat检查是否为文件（而不是文件夹）
					if info, err := os.Stat(path); err == nil && !info.IsDir() {
						// 确认是文件才进行格式验证
						if err := validatePath(path); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						m.act.cur.path = absPath(path)
						m.upStep = stepAskMore
						return nil
					}
					// 如果是文件夹或无法访问，什么都不做，让filepicker正常处理
				}

				// 检查禁用文件选择
				if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
					// 只对文件显示格式错误，文件夹不需要格式验证
					if info, err := os.Stat(path); err == nil && !info.IsDir() {
						m.act.filePickerErr = errors.New(path + " 文件格式不支持")
						return clearFilePickerErrorAfter(3 * time.Second)
					}
				}
			}

			return tea.Batch(pathCmd, fpCmd)
		case stepAskMore:
			var cmd tea.Cmd
			m.moreList, cmd = m.moreList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.moreList.SelectedItem().(listItem); ok {
					title := it.title
					if strings.HasPrefix(title, "是") {
						// 保存当前版本，继续添加
						m.act.versions = append(m.act.versions, m.act.cur)
						next := fmt.Sprintf("v%d.0", len(m.act.versions)+1)
						m.act.cur = versionItem{}
						m.inpVersion.SetValue(next)
						m.inpCover.SetValue("")
						m.inpIntro.SetValue("")
						m.upStep = stepVersion
						return nil
					}
					// 否，进入确认
					m.act.versions = append(m.act.versions, m.act.cur)
					m.upStep = stepConfirm
					m.act.confirming = true
					return nil
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				m.upStep = stepPath
				m.act.useFilePicker = true
				return nil
			}
			return cmd
		case stepConfirm:
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					m.running = true
					// 启动多版本上传
					return runUploadActionMulti(m.act.u, m.act.versions)
				case "esc":
					// 返回"是否继续添加版本"
					m.act.confirming = false
					m.upStep = stepAskMore
					return nil
				}
			}
			return nil
		default:
			return nil
		}
	case actionLsModel:
		// 直接调用现有命令，需先选择类型
		if m.act.u.typ == "" {
			var cmd tea.Cmd
			m.typeList, cmd = m.typeList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.typeList.SelectedItem().(listItem); ok {
					m.act.u.typ = it.title
					// 选择完类型后，进入是否公开的选择（toggle）
					return nil
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				m.step = mainStepMenu
				m.act = actionInputs{}
				return nil
			}
			return cmd
		}
		// 在 lsPublic 上增加空格切换/Enter 执行
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case " ":
				m.act.lsPublic = !m.act.lsPublic
				return nil
			case "enter":
				m.running = true
				return runListModelsWithPublic(m.act.u.typ, m.act.lsPublic)
			case "esc":
				m.act.u.typ = ""
				return nil
			}
		}
		return nil
	case actionLsFiles:
		// 选择类型 -> 输入 name -> 输入 ext（可选，Esc跳过）-> 是否树形/是否公开 -> 执行
		if m.act.u.typ == "" {
			var cmd tea.Cmd
			m.typeList, cmd = m.typeList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.typeList.SelectedItem().(listItem); ok {
					m.act.u.typ = it.title
					return m.inpName.Focus()
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				m.step = mainStepMenu
				m.act = actionInputs{}
				return nil
			}
			return cmd
		}
		if m.act.u.name == "" {
			var cmd tea.Cmd
			m.inpName, cmd = m.inpName.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if err := validateName(m.inpName.Value()); err != nil {
					m.err = err
					return nil
				}
				m.act.u.name = m.inpName.Value()
				return m.inpExt.Focus()
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 返回选择类型
				m.act.u.typ = ""
				return nil
			}
			return cmd
		}
		// 输入 ext（可空）。Enter 确认，Esc 跳过
		if !m.act.lfExtDone {
			var cmd tea.Cmd
			m.inpExt, cmd = m.inpExt.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					// 回车：若为空则跳过，否则确认
					m.act.lfExt = m.inpExt.Value()
					m.act.lfExtDone = true
				}
			}
			// 当确认后，进入开关选择
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				return nil
			}
			return cmd
		}

		// 在 lfTree/lfPublic 开关：空格切换树形，tab 切换到 public，enter 执行，esc 返回
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case " ":
				m.act.lfTree = !m.act.lfTree
				return nil
			case "tab":
				m.act.lfPublic = !m.act.lfPublic
				return nil
			case "enter":
				m.running = true
				return runListFilesWithOptions(m.act.u.typ, m.act.u.name, m.act.lfExt, m.act.lfTree, m.act.lfPublic)
			case "esc":
				// 返回 ext 输入
				m.act.lfExtDone = false
				return m.inpExt.Focus()
			}
		}
		return nil
	case actionRmModel:
		// 选择类型 -> 输入 name -> 执行
		if m.act.u.typ == "" {
			var cmd tea.Cmd
			m.typeList, cmd = m.typeList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.typeList.SelectedItem().(listItem); ok {
					m.act.u.typ = it.title
					return m.inpName.Focus()
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				m.step = mainStepMenu
				m.act = actionInputs{}
				return nil
			}
			return cmd
		}
		if m.act.u.name == "" {
			var cmd tea.Cmd
			m.inpName, cmd = m.inpName.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if err := validateName(m.inpName.Value()); err != nil {
					m.err = err
					return nil
				}
				m.act.u.name = m.inpName.Value()
				m.running = true
				return runRemoveModel(m.act.u.typ, m.act.u.name)
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 返回选择类型
				m.act.u.typ = ""
				return nil
			}
			return cmd
		}
		return nil
	default:
		return nil
	}
}

// 渲染当前动作视图
func (m *mainModel) renderActionView() string {
	switch m.currentAction {
	case actionUpload:
		switch m.upStep {
		case stepName:
			return m.titleStyle.Render("上传 · Step 1/8 · 模型名称") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepType:
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 2/8 · 选择模型类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepVersion:
			return m.titleStyle.Render("上传 · Step 3/8 · 版本名称（默认 v1.0）") + "\n\n" + m.inpVersion.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepBase:
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.baseList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 4/8 · Base Model（可选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("(可选) 选择后 Enter，或直接 Enter 跳过")
		case stepCover:
			return m.titleStyle.Render("上传 · Step 5/8 · Cover（可选；多个用 ; 分隔）") + "\n\n" + m.inpCover.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepIntro:
			return m.titleStyle.Render("上传 · Step 6/8 · 模型介绍（可为空，回车跳过）") + "\n\n" + m.inpIntro.View() + "\n" + m.hintStyle.Render("确认：Enter（可空），返回：Esc，退出：q")
		case stepPath:
			var content strings.Builder
			content.WriteString(m.titleStyle.Render("上传 · Step 7/8 · 选择文件"))
			content.WriteString("\n\n")

			// 路径输入框部分
			pathInputLabel := "路径输入："
			if m.act.pathInputFocused {
				pathInputLabel = m.titleStyle.Render("► 路径输入：（当前焦点）")
			} else {
				pathInputLabel = m.hintStyle.Render("路径输入：")
			}
			content.WriteString(pathInputLabel)
			content.WriteString("\n")
			content.WriteString(m.inpPath.View())
			content.WriteString("\n\n")

			// 文件选择器部分标题
			filePickerLabel := "文件选择器："
			if !m.act.pathInputFocused {
				filePickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点）")
			} else {
				filePickerLabel = m.hintStyle.Render("文件选择器：")
			}
			content.WriteString(filePickerLabel)
			content.WriteString("\n")

			// 状态显示部分（类似官方示例）
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
			} else if m.selectedFile == "" {
				content.WriteString("选择一个文件:")
			} else {
				content.WriteString("已选择文件: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
			}
			content.WriteString("\n")

			// 文件选择器部分
			content.WriteString(m.filepicker.View())
			content.WriteString("\n")

			// 操作提示 - 根据焦点状态给出不同的提示
			if m.act.pathInputFocused {
				content.WriteString(m.hintStyle.Render("Enter确认路径（目录会自动切换），Tab切换到文件选择器，Ctrl+R强制刷新，返回：Esc"))
			} else {
				// 检查路径输入框和filepicker目录是否同步
				inputPath := strings.TrimSpace(m.inpPath.Value())
				inputDir := inputPath
				if !strings.HasSuffix(inputDir, "/") && inputDir != "" {
					inputDir = inputDir + "/"
				}
				currentDir := m.filepicker.CurrentDirectory
				if !strings.HasSuffix(currentDir, "/") {
					currentDir = currentDir + "/"
				}

				if inputPath != "" && inputDir != currentDir {
					content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Tab同步目录，Ctrl+R强制刷新，返回：Esc"))
				} else {
					content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Tab切换输入，Ctrl+R刷新，返回：Esc"))
				}
			}

			return content.String()
		case stepAskMore:
			var b strings.Builder
			b.WriteString(m.titleStyle.Render("上传 · Step 8/8 · 是否继续添加版本？"))
			b.WriteString("\n\n")
			if len(m.act.versions) > 0 {
				b.WriteString("已添加版本：\n")
				for i, v := range m.act.versions {
					b.WriteString(fmt.Sprintf("  - [%d] %s  base=%s  cover=%s  path=%s\n", i+1, dash(v.version), dash(v.base), dash(v.cover), dash(v.path)))
				}
				b.WriteString("\n")
			}
			cur := m.act.cur
			b.WriteString("当前版本：\n")
			b.WriteString(fmt.Sprintf("  - %s  base=%s  cover=%s  path=%s\n\n", dash(cur.version), dash(cur.base), dash(cur.cover), dash(cur.path)))
			// 在下方渲染选择列表
			return b.String() + "\n" + m.moreList.View() + "\n" + m.hintStyle.Render("Enter 确认选择，Esc 返回上一页")
		case stepConfirm:
			var b strings.Builder
			b.WriteString(m.titleStyle.Render("上传 · 确认所有版本"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("模型名称：%s\n类型：%s\n\n", m.act.u.name, m.act.u.typ))
			for i, v := range m.act.versions {
				b.WriteString(fmt.Sprintf("[%d] 版本=%s  base=%s\n", i+1, dash(v.version), dash(v.base)))
				b.WriteString(fmt.Sprintf("cover=%s\npath=%s\nintro=%s\n\n", dash(v.cover), dash(v.path), dash(v.intro)))
			}
			b.WriteString(m.hintStyle.Render("按 Enter 开始上传；Esc 返回上一步；q 退出"))
			return b.String()
		}
		return ""
	case actionLsModel:
		if m.act.u.typ == "" {
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("列出模型 · 选择类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		// 公共开关提示
		publicStr := "否"
		if m.act.lsPublic {
			publicStr = "是"
		}
		return m.titleStyle.Render("列出模型 · 是否仅显示公开模型？") + "\n\n" +
			"当前设置：" + publicStr + "  （切换：空格，确认：Enter，返回：Esc）"
	case actionLsFiles:
		if m.act.u.typ == "" {
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("查看文件 · 选择类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		if m.act.u.name == "" {
			return m.titleStyle.Render("查看文件 · 输入模型名") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		if !m.act.lfExtDone {
			return m.titleStyle.Render("查看文件 · 过滤扩展名（可选）") + "\n\n" + m.inpExt.View() + "\n" + m.hintStyle.Render("确认：Enter（输入为空则跳过），退出：q")
		}
		// 显示开关状态
		treeStr := "否"
		if m.act.lfTree {
			treeStr = "是"
		}
		pubStr := "否"
		if m.act.lfPublic {
			pubStr = "是"
		}
		return m.titleStyle.Render("查看文件 · 选项") + "\n\n" +
			"树形显示：" + treeStr + "  （切换：空格）\n" +
			"仅公开：" + pubStr + "  （切换：Tab）\n\n" +
			m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
	case actionRmModel:
		if m.act.u.typ == "" {
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("删除模型 · 选择类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		if m.act.u.name == "" {
			return m.titleStyle.Render("删除模型 · 输入模型名") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		return m.hintStyle.Render("执行中…（返回：Esc）")
	default:
		return ""
	}
}

// 登录校验 + 保存
func loginCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.AuthDomain, apiKey)
		if _, err := client.UserInfo(); err != nil {
			return loginDoneMsg{ok: false, err: err}
		}
		if err := lib.NewSfFolder().SaveKey(apiKey); err != nil {
			return loginDoneMsg{ok: false, err: err}
		}
		return loginDoneMsg{ok: true}
	}
}

// 运行 whoami（复用现有子命令）
func runWhoami(apiKey string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{}
		if apiKey != "" {
			args = append(args, "--api_key", apiKey)
		}
		args = append(args, "whoami")
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// 运行 logout
func runLogout() tea.Cmd {
	return func() tea.Msg {
		err := lib.NewSfFolder().RemoveKey()
		if err != nil {
			return actionDoneMsg{out: "", err: err}
		}
		return actionDoneMsg{out: "Logged out successfully\n", err: nil}
	}
}

// 运行上传（调用现有 upload 子命令）
// obsolete single-version uploader removed (replaced by runUploadActionMulti)

// 多版本上传
func runUploadActionMulti(u uploadInputs, versions []versionItem) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan tea.Msg, 64)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer close(ch)

			args := config.NewArgument()
			args.Type = u.typ
			args.Name = u.name

			if err := checkType(args, true); err != nil {
				ch <- actionDoneMsg{err: err}
				return
			}
			if err := checkName(args, true); err != nil {
				ch <- actionDoneMsg{err: err}
				return
			}

			apiKey, err := lib.NewSfFolder().GetKey()
			if err != nil || apiKey == "" {
				ch <- actionDoneMsg{out: "", err: fmt.Errorf("未登录或缺少 API Key，请先登录")}
				return
			}
			client := lib.NewClient(meta.DefaultDomain, apiKey)

			// 逐版本上传
			mvList := make([]*lib.ModelVersion, 0, len(versions))
			for i, v := range versions {
				// 准备 FileToUpload
				st, err := os.Stat(v.path)
				if err != nil {
					ch <- actionDoneMsg{err: err}
					return
				}
				if st.IsDir() {
					ch <- actionDoneMsg{err: fmt.Errorf("仅支持文件上传: %s", v.path)}
					return
				}

				relPath, err := filepath.Rel(filepath.Dir(v.path), v.path)
				if err != nil {
					ch <- actionDoneMsg{err: err}
					return
				}
				f := &lib.FileToUpload{Path: filepath.ToSlash(v.path), RelPath: filepath.ToSlash(relPath), Size: st.Size()}

				sha256sum, md5Hash, err := calculateHash(f.Path)
				if err != nil {
					ch <- actionDoneMsg{err: err}
					return
				}
				f.Signature = sha256sum

				ossCert, err := client.OssSign(sha256sum, u.typ)
				if err != nil {
					ch <- actionDoneMsg{err: err}
					return
				}
				fileIndex := fmt.Sprintf("%d/%d", i+1, len(versions))
				fileRecord := ossCert.Data.File

				if fileRecord.Id == 0 {
					storage := ossCert.Data.Storage
					ossClient, err := lib.NewAliOssStorageClient(storage.Endpoint, storage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
					if err != nil {
						ch <- actionDoneMsg{err: err}
						return
					}
					_, err = ossClient.UploadFileCtx(ctx, f, fileRecord.ObjectKey, fileIndex, func(consumed, total int64) {
						select {
						case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: consumed, total: total, verIdx: i}:
						default:
						}
					})
					if err != nil {
						if errors.Is(err, context.Canceled) {
							ch <- actionDoneMsg{out: "上传已取消\n", err: nil}
							return
						}
						ch <- actionDoneMsg{err: err}
						return
					}
					if _, err = client.CommitFileV2(f.Signature, fileRecord.ObjectKey, md5Hash, u.typ); err != nil {
						ch <- actionDoneMsg{err: err}
						return
					}
				} else {
					f.Id = fileRecord.Id
					f.RemoteKey = fileRecord.ObjectKey
					select {
					case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: f.Size, total: f.Size, verIdx: i}:
					default:
					}
				}

				// 组装版本
				mv := &lib.ModelVersion{Version: v.version, BaseModel: v.base, Introduction: v.intro, Public: false, Sign: f.Signature, Path: v.path}
				if v.cover != "" {
					mv.CoverUrls = strings.Split(v.cover, ";")
				}
				mvList = append(mvList, mv)
			}

			// 提交模型（所有版本）
			if _, err := client.CommitModelV2(u.name, u.typ, mvList); err != nil {
				ch <- actionDoneMsg{err: err}
				return
			}
			ch <- actionDoneMsg{out: "Uploaded successfully\n", err: nil}
		}()
		return uploadStartMsg{ch: ch, cancel: cancel}
	}
}

// 运行列出模型
func runListModels(typ string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{"model", "ls", "--type", typ}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// 运行列出模型（含 public 开关）
func runListModelsWithPublic(typ string, public bool) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{"model", "ls", "--type", typ}
		if public {
			args = append(args, "--public")
		}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// 运行列出文件
func runListFiles(typ, name string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{"model", "ls-files", "--type", typ, "--name", name}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// 运行列出文件（带 ext/tree/public）
func runListFilesWithOptions(typ, name, ext string, tree, public bool) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{"model", "ls-files", "--type", typ, "--name", name}
		if ext != "" {
			args = append(args, "--ext", ext)
		}
		if tree {
			args = append(args, "--tree")
		}
		if public {
			args = append(args, "--public")
		}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// 运行删除模型
func runRemoveModel(typ, name string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{"model", "rm", "--type", typ, "--name", name}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
	}
}

// CLI 入口
func MainTUI(c *cli.Context) error {
	p := tea.NewProgram(newMainModel(), tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return err
	}
	// 当输出页按 Enter 返回菜单时，重新进入主界面
	if _, ok := model.(mainModel); ok {
		return nil
	}
	return nil
}
