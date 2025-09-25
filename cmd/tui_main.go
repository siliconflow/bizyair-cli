package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// 清除filepicker错误的消息
type clearFilePickerErrorMsg struct{}

// 菜单项
type menuEntry struct {
	listItem
	key actionKind
}

// 上传所需输入
type uploadInputs struct {
	typ   string
	name  string
	path  string
	base  string
	cover string
}

// 针对不同动作的输入组件（尽量精简，必要时逐步扩展）
type actionInputs struct {
	// 通用
	confirming bool

	// Upload
	u uploadInputs

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
	inpName  textinput.Model
	inpPath  textinput.Model
	inpCover textinput.Model
	// 其他输入
	inpExt textinput.Model

	//文件选择器
	filepicker   filepicker.Model
	selectedFile string

	// 动作输入
	act actionInputs

	// 运行态
	running bool
	output  string
	sp      spinner.Model

	// 样式
	titleStyle lipgloss.Style
	hintStyle  lipgloss.Style
	panelStyle lipgloss.Style
	btnStyle   lipgloss.Style

	// 品牌
	logo           string
	logoStyle      lipgloss.Style
	smallLogoStyle lipgloss.Style
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

	// 输入框
	inApi := textinput.New()
	inApi.Placeholder = "请输入 API Key"

	inName := textinput.New()
	inName.Placeholder = "请输入 name（字母/数字/下划线/短横线）"

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

	return mainModel{
		step:       mainStepHome,
		menu:       menuList,
		inpApi:     inApi,
		typeList:   tp,
		baseList:   bl,
		inpName:    inName,
		inpPath:    inPath,
		inpCover:   inCover,
		inpExt:     inExt,
		filepicker: fp,
		titleStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#36A3F7")),
		hintStyle:  lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		panelStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2),
		btnStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#04B575")).Padding(0, 1).Bold(true),
		sp:         sp,
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

func (m mainModel) Init() tea.Cmd { return tea.Batch(m.sp.Tick, m.filepicker.Init()) }

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

	m.act.u.path = absPath(path)
	m.selectedFile = path
	m.act.filePickerErr = nil
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
						return m, nil
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
		return m, nil
	case clearFilePickerErrorMsg:
		m.act.filePickerErr = nil
		return m, nil
	default:
		// 转给 spinner
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		if cmd != nil {
			return m, cmd
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

	// 全局运行中覆盖视图：无论当前处于哪个步骤，统一展示等待界面
	if m.running {
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
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// 让内部组件也能拿到 key 事件
			_ = msg
		}
		// 依序：type -> name -> path -> base -> cover -> confirm -> run
		if m.act.u.typ == "" {
			m.typeList, cmd = m.typeList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.typeList.SelectedItem().(listItem); ok {
					m.act.u.typ = it.title
					return m.inpName.Focus()
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 返回上级菜单
				m.step = mainStepMenu
				m.act = actionInputs{}
				m.selectedFile = ""
				m.filepicker.Path = ""
				return nil
			}
			return cmd
		}
		if m.act.u.name == "" {
			m.inpName, cmd = m.inpName.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if err := validateName(m.inpName.Value()); err != nil {
					m.err = err
					return nil
				}
				m.act.u.name = m.inpName.Value()
				m.act.useFilePicker = true
				m.act.pathInputFocused = false // 默认filepicker有焦点

				// 设置路径输入框的初始值为filepicker的当前目录
				m.inpPath.SetValue(m.filepicker.CurrentDirectory + "/")

				// 重新初始化filepicker以确保它正确读取目录，并让路径输入框失去焦点
				m.inpPath.Blur()
				return m.filepicker.Init()
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 回到选择类型
				m.act.u = uploadInputs{}
				return nil
			}
			return cmd
		}
		if m.act.u.path == "" {
			// 混合模式：路径输入框 + 文件选择器，类似官方示例
			var pathCmd, fpCmd tea.Cmd

			// 处理按键事件
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "esc":
					// 返回上一步
					m.act.u.name = ""
					m.act.useFilePicker = false
					m.act.pathInputFocused = false
					m.act.filePickerErr = nil
					m.filepicker.Path = ""
					return m.inpName.Focus()
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
						// 确认是文件才进行格式验证和选择
						if err := m.validateAndSetPath(path); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
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
		}
		if m.act.u.base == "" && !m.act.confirming {
			m.baseList, cmd = m.baseList.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				if it, ok := m.baseList.SelectedItem().(listItem); ok {
					if it.title != "(跳过)" {
						if !meta.SupportedBaseModels[it.title] {
							m.err = fmt.Errorf("不支持的 Base Model: %s", it.title)
							return nil
						}
						m.act.u.base = it.title
					}
					return m.inpCover.Focus()
				}
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 返回 path
				m.act.u.path = ""
				return m.inpPath.Focus()
			}
			return cmd
		}
		if !m.act.confirming {
			m.inpCover, cmd = m.inpCover.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
				m.act.u.cover = m.inpCover.Value()
				m.act.confirming = true
			} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
				// 返回 base 选择
				m.act.u.base = ""
				return nil
			}
			return cmd
		}
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
			m.running = true
			return runUploadAction(m.act.u)
		} else if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
			// 返回 cover 输入
			m.act.confirming = false
			return m.inpCover.Focus()
		}
		return nil
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
		if m.act.u.typ == "" {
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 1/6 · 选择模型类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		if m.act.u.name == "" {
			return m.titleStyle.Render("上传 · Step 2/6 · Name") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		if m.act.u.path == "" {
			var content strings.Builder
			content.WriteString(m.titleStyle.Render("上传 · Step 3/6 · 选择文件"))
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
		}
		if m.act.u.base == "" && !m.act.confirming {
			if m.height > 0 {
				h := m.height - 10
				if h < 5 {
					h = 5
				}
				m.baseList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 4/6 · Base Model（可选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("(可选) 选择后 Enter，或直接 Enter 跳过")
		}
		if !m.act.confirming {
			return m.titleStyle.Render("上传 · Step 5/6 · Cover（可选，多个用 ; 分隔）") + "\n\n" + m.inpCover.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		}
		summary := fmt.Sprintf("- type: %s\n- name: %s\n- path: %s\n- base: %s\n- cover: %s", dash(m.act.u.typ), dash(m.act.u.name), dash(m.act.u.path), dash(m.act.u.base), dash(m.act.u.cover))
		return m.titleStyle.Render("上传 · Step 6/6 · 确认") + "\n\n" + summary + "\n\n" + m.hintStyle.Render("按 Enter 开始，q 退出")
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
func runUploadAction(in uploadInputs) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{
			"upload",
			"--type", in.typ,
			"--name", in.name,
			"--path", in.path,
		}
		if in.base != "" {
			args = append(args, "--base", in.base)
		}
		if in.cover != "" {
			args = append(args, "--cover", in.cover)
		}
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: err}
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
