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
	"sync"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
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

// 模型列表加载完成消息
type modelListLoadedMsg struct {
	models []*lib.BizyModelInfo
	total  int
	err    error
}

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

	// 模型列表 table
	modelTable       table.Model
	modelList        []*lib.BizyModelInfo
	modelListTotal   int
	loadingModelList bool

	// 模型列表列宽（随窗口自适应，供截断使用）
	nameColWidth int
	fileColWidth int

	// 模型详情
	viewingModelDetail bool
	modelDetail        *lib.BizyModelDetail
	loadingModelDetail bool

	// 删除确认
	confirmingDelete bool
	deleteTargetId   int64
	deleteTargetName string

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

	// 外框内边距（用于全屏外边框内的留白）
	framePadX int
	framePadY int

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
		menuEntry{listItem{title: "我的模型", desc: "浏览我的模型"}, actionLsModel},
		menuEntry{listItem{title: "当前账户信息", desc: "显示 whoami"}, actionWhoami},
		menuEntry{listItem{title: "退出登录", desc: "清除本地 API Key"}, actionLogout},
		menuEntry{listItem{title: "退出程序", desc: "离开 BizyAir CLI"}, actionExit},
	}

	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#04B575")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)

	menuList := list.New(mItems, d, 30, len(mItems)*8 )
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
	bItems := []list.Item{}
	for _, b := range bases {
		bItems = append(bItems, listItem{title: b})
	}
	bl := list.New(bItems, d, 30, 12)
	bl.Title = "选择 Base Model（必选）"

	// 是否添加更多版本选择
	moreItems := []list.Item{listItem{title: "是，继续添加版本"}, listItem{title: "否，进入确认"}}
	ml := list.New(moreItems, d, 30, 12)
	ml.Title = "是否继续添加版本？"
	ml.SetShowStatusBar(false)
	ml.SetShowPagination(false)

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

	// 模型列表 table
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "名称", Width: 25},
		{Title: "类型", Width: 12},
		{Title: "版本数", Width: 8},
		{Title: "Base Model", Width: 15},
		{Title: "文件名", Width: 60},
	}

	modelTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#36A3F7"))
	// 移除默认左右内边距，避免列宽之外的额外宽度
	s.Cell = s.Cell.PaddingLeft(0).PaddingRight(0)
	s.Header = s.Header.PaddingLeft(0).PaddingRight(0)
	s.Selected = s.Selected.PaddingLeft(0).PaddingRight(0)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
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
		logoStyle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")),
		smallLogoStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EC4899")),
		// 外框默认内边距：左右 2、上下 1
		framePadX: 2,
		framePadY: 1,
	}

	// 启动即合并首页与菜单页：若已登录则直接进入菜单；否则进入登录
	if key, err := lib.NewSfFolder().GetKey(); err == nil && key != "" {
		m.loggedIn = true
		m.apiKey = key
		m.step = mainStepMenu
	} else {
		m.step = mainStepLogin
	}

	return m
}

// 计算指定屏幕宽高下的内层区域尺寸（去除外框边框与外框内边距）
func (m *mainModel) computeInnerSizeFor(totalW, totalH int) (int, int) {
	iw := totalW - 2 - m.framePadX*2
	ih := totalH - 2 - m.framePadY*2
	if iw < 1 {
		iw = 1
	}
	if ih < 1 {
		ih = 1
	}
	return iw, ih
}

// 当前内层尺寸
func (m *mainModel) innerSize() (int, int) {
	return m.computeInnerSizeFor(m.width, m.height)
}

// 将字符串裁剪/填充为固定宽度（兼容 ANSI 宽度）
func clipToWidth(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

// 渐变外框渲染：在整屏绘制蓝紫(#8B5CF6)->粉色(#EC4899)的边框，并在内侧使用内边距包裹内容
func (m *mainModel) renderFrame(inner string) string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return inner
	}

	iw, ih := m.innerSize()
	padX, padY := m.framePadX, m.framePadY

	// 拆分内层内容成行，并限制到内层高度
	lines := strings.Split(inner, "\n")
	contentLines := make([]string, ih)
	for i := 0; i < ih; i++ {
		if i < len(lines) {
			contentLines[i] = clipToWidth(lines[i], iw)
		} else {
			contentLines[i] = clipToWidth("", iw)
		}
	}

	// 渐变颜色（纵向）：顶端为紫色，底部为粉色
	sr, sg, sb := hexToRGB("#8B5CF6")
	er, eg, eb := hexToRGB("#EC4899")

	var b strings.Builder

	// 顶部边框（使用顶端颜色）
	if h >= 1 {
		tColor := rgbToHex(sr, sg, sb)
		tStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tColor))
		if w >= 2 {
			b.WriteString(tStyle.Render("╭" + strings.Repeat("─", w-2) + "╮"))
		} else {
			b.WriteString(tStyle.Render("╭"))
		}
		b.WriteString("\n")
	}

	// 中间区域行（包含左右竖边框、内边距与内容）
	innerStartY := padY
	innerEndY := padY + ih
	for y := 1; y <= h-2; y++ {
		// y 从 1 到 h-2
		t := 0.0
		if h > 1 {
			t = float64(y) / float64(h-1)
		}
		r := lerpInt(sr, er, t)
		g := lerpInt(sg, eg, t)
		bl := lerpInt(sb, eb, t)
		c := rgbToHex(r, g, bl)
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(c))

		// 内部空白或内容
		middle := ""
		if w >= 2 {
			if y-1 < innerStartY || y-1 >= innerEndY {
				// 处于上/下内边距区域
				middle = strings.Repeat(" ", w-2)
			} else {
				row := y - 1 - innerStartY
				if row >= 0 && row < len(contentLines) {
					middle = strings.Repeat(" ", padX) + clipToWidth(contentLines[row], iw) + strings.Repeat(" ", padX)
				} else {
					middle = strings.Repeat(" ", w-2)
				}
			}
		}

		if w >= 2 {
			b.WriteString(s.Render("│") + middle + s.Render("│"))
		} else {
			b.WriteString(s.Render("│"))
		}
		b.WriteString("\n")
	}

	// 底部边框（使用底部颜色）
	if h >= 2 {
		bColor := rgbToHex(er, eg, eb)
		bStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(bColor))
		if w >= 2 {
			b.WriteString(bStyle.Render("╰" + strings.Repeat("─", w-2) + "╯"))
		} else {
			b.WriteString(bStyle.Render("╰"))
		}
	}

	return b.String()
}

// 动态调整模型列表列宽以占满面板宽度
func (m *mainModel) resizeModelTable(totalWidth int) {
	if totalWidth <= 0 {
		return
	}
	// 计算实际可用宽度：
	// 顶层有 panel 边框与 Padding(1,2)，以及 header 行等文本宽度与 PlaceHorizontal 不影响 panel 内宽；
	// 为安全起见，保守扣减 8 列，避免全屏时溢出（边框2 + 左右内边距4 + 余量2）
	usable := totalWidth - 8
	if usable < 60 {
		usable = 60
	}

	// 固定列（ID、类型、版本数、Base Model）分配最小宽度
	idW := 10
	typeW := 12
	verW := 8
	baseMin := 15

	// 余量给可变列（名称、文件名、Base Model 部分可增长）
	// 初始给名称 20，文件名 40，Base 最小 15
	remaining := usable - (idW + typeW + verW)
	if remaining < 20+15+20 {
		// 最小兜底
		remaining = 20 + 15 + 20
	}

	// 比例分配：名称 28%，Base 18%，文件名 54%（更偏向长文件名，减少右侧溢出）
	nameW := int(float64(remaining) * 0.28)
	baseW := int(float64(remaining) * 0.18)
	fileW := remaining - nameW - baseW
	if baseW < baseMin {
		// 不足则从文件名列借
		deficit := baseMin - baseW
		baseW = baseMin
		if fileW > deficit+10 {
			fileW -= deficit
		} else if nameW > deficit+10 {
			nameW -= deficit
		}
	}

	// 保存供截断使用（预留 1 列作为冗余，避免字符宽度误差换行）
	m.nameColWidth = maxInt(5, nameW-1)
	m.fileColWidth = maxInt(8, fileW-1)

	cols := []table.Column{
		{Title: "ID", Width: idW},
		{Title: "名称", Width: nameW},
		{Title: "类型", Width: typeW},
		{Title: "版本数", Width: verW},
		{Title: "Base Model", Width: baseW},
		{Title: "文件名", Width: fileW},
	}
	m.modelTable.SetColumns(cols)

	// 列宽变更后重建行，使内容按新宽度截断
	if len(m.modelList) > 0 {
		m.rebuildModelTableRows()
	}
}

// 根据当前列宽重建表格行
func (m *mainModel) rebuildModelTableRows() {
	rows := []table.Row{}
	for _, model := range m.modelList {
		versionCount := fmt.Sprintf("%d", len(model.Versions))
		baseModels := []string{}
		fileNames := []string{}
		for _, version := range model.Versions {
			if version.BaseModel != "" {
				baseModels = append(baseModels, version.BaseModel)
			}
			if version.FileName != "" {
				fileNames = append(fileNames, version.FileName)
			}
		}
		baseModelStr := "-"
		if len(baseModels) > 0 {
			uniqueBaseModels := uniqueStrings(baseModels)
			if len(uniqueBaseModels) == 1 {
				baseModelStr = uniqueBaseModels[0]
			} else {
				baseModelStr = fmt.Sprintf("%s (共%d个)", uniqueBaseModels[0], len(uniqueBaseModels))
			}
		}
		fileNameStr := "-"
		if len(fileNames) > 0 {
			if len(fileNames) == 1 {
				fileNameStr = fileNames[0]
			} else {
				fileNameStr = fmt.Sprintf("%s (共%d个)", fileNames[0], len(fileNames))
			}
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", model.Id),
			truncateString(model.Name, maxInt(5, m.nameColWidth)),
			model.Type,
			versionCount,
			baseModelStr,
			truncateString(fileNameStr, maxInt(8, m.fileColWidth)),
		})
	}
	m.modelTable.SetRows(rows)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

// 验证并设置路径的辅助方法
func (m *mainModel) validateAndSetPath(path string) error {
	// 验证文件扩展名
	supportedExts := []string{".safetensors"}
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

	return nil
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// 使用外框后的内层宽高
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

		// 进度条宽度自适应屏幕
		m.progress.Width = innerW - 6
		if m.progress.Width < 10 {
			m.progress.Width = 10
		}

		// 多版本进度条宽度自适应
		if len(m.verProgress) > 0 {
			for i := range m.verProgress {
				m.verProgress[i].Width = innerW - 6
				if m.verProgress[i].Width < 10 {
					m.verProgress[i].Width = 10
				}
			}
		}

		// 设置filepicker的高度
		if innerH > 15 {
			m.filepicker.SetHeight(innerH - 15)
		} else {
			m.filepicker.SetHeight(5)
		}

		// 动态调整模型表格列宽以占满可用宽度
		m.resizeModelTable(innerW)
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
						m.loadingModelList = true
						// 直接加载模型列表
						return m, loadModelList(m.apiKey)

					}
				}
			case mainStepAction:
				// 如果在模型列表页，Enter 进入详情（删除确认期间禁止）
				if m.currentAction == actionLsModel && !m.loadingModelList && len(m.modelList) > 0 && !m.viewingModelDetail && !m.confirmingDelete {
					selectedRow := m.modelTable.SelectedRow()
					if len(selectedRow) > 0 {
						// 第一列为 ID
						idStr := selectedRow[0]
						if idStr != "" {
							if id64, err := strconv.ParseInt(idStr, 10, 64); err == nil {
								// 在 modelList 中查找该 ID 对应的模型（稳妥）
								var targetId int64 = id64
								for _, mInfo := range m.modelList {
									if mInfo.Id == targetId {
										m.loadingModelDetail = true
										m.viewingModelDetail = true
										return m, loadModelDetail(m.apiKey, targetId)
									}
								}
							}
						}
					}
				}
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
	case modelListLoadedMsg:
		m.loadingModelList = false
		if msg.err != nil {
			m.err = msg.err
			m.step = mainStepOutput
			m.output = fmt.Sprintf("加载模型列表失败: %v", msg.err)
			return m, nil
		}
		m.modelList = msg.models
		m.modelListTotal = msg.total
		// 更新表格数据（使用动态列宽截断）
		m.rebuildModelTableRows()
		return m, nil
	case modelDetailLoadedMsg:
		m.loadingModelDetail = false
		if msg.err != nil {
			m.err = msg.err
			// 回到列表视图但保持列表数据
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
		// 确保处于列表态
		m.viewingModelDetail = false
		m.modelDetail = nil
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// 成功后刷新列表
		m.loadingModelList = true
		return m, loadModelList(m.apiKey)
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

	// 更新 spinner（当加载模型列表时）
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

	var viewStr string

	// 合并 Logo + 菜单：在菜单页顶部渲染大 Logo 渐变
	header := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, m.smallLogoStyle.Render("BizyAir CLI"))

	if m.err != nil && m.step != mainStepOutput {
		defer func() { m.err = nil }()
		viewStr = header + "\n" + panel.Render(m.titleStyle.Render("错误")+"\n"+fmt.Sprintf("%v", m.err)+"\n\n"+m.hintStyle.Render("按任意键返回继续…"))
		return m.renderFrame(viewStr)
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

			// 构造进度显示（单/多版本）
			var progressSection strings.Builder
			if len(m.verProgress) > 0 {
				// 顶部显示当前文件
				if m.uploadProg.total > 0 {
					progressSection.WriteString(fmt.Sprintf("当前: (%s) %s\n", m.uploadProg.fileIndex, m.uploadProg.fileName))
				} else {
					progressSection.WriteString("准备上传…\n")
				}
				for i := range m.verProgress {
					versionLabel := ""
					if i >= 0 && i < len(m.act.versions) {
						versionLabel = m.act.versions[i].version
					}
					consumed := m.verConsumed[i]
					total := m.verTotal[i]
					percent := 0.0
					if total > 0 {
						percent = float64(consumed) / float64(total)
					}
					bar := m.verProgress[i].View()
					prefix := "  "
					if i == m.uploadProg.verIdx {
						prefix = "▶ "
					}
					progressSection.WriteString(fmt.Sprintf("%s[%d/%d] 版本=%s\n", prefix, i+1, len(m.verProgress), dash(versionLabel)))
					if total > 0 {
						progressSection.WriteString(fmt.Sprintf("%s%s %.1f%% (%s/%s)\n", prefix, bar, percent*100, formatBytes(consumed), formatBytes(total)))
					} else {
						progressSection.WriteString(fmt.Sprintf("%s%s\n", prefix, bar))
					}
				}
			} else {
				// 单版本回退显示
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
				progressSection.WriteString(fileLine + "\n" + progLine)
			}

			content := m.titleStyle.Render("上传中 · 请稍候") + "\n\n" + summary + "\n\n" + progressSection.String()
			viewStr = header + "\n" + panel.Render(content)
			return m.renderFrame(viewStr)
		}
		spin := m.sp.View()
		viewStr = header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…")
		return m.renderFrame(viewStr)
	}

	switch m.step {
	case mainStepLogin:
		viewStr = header + "\n" + panel.Render(m.titleStyle.Render("登录 · 请输入 API Key")+"\n\n"+m.inpApi.View()+"\n"+m.hintStyle.Render("确认：Enter，返回：Esc，退出：q"))
		return m.renderFrame(viewStr)
	case mainStepMenu:
		// 构造顶部大 Logo 渐变
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

		// 菜单高度为内层高度 - logo 占用行数 - 标题与边距
		menuTop := strings.Count(logoStr, "\n") + 2 // 预估顶部占用
		h := innerH - menuTop - 6
		if h < 6 {
			h = 6
		}
		m.menu.SetHeight(h)

		// 取消 panel 边框，仅保留外层全屏渐变外框
		viewStr = logoStr + "\n\n" + m.titleStyle.Render("功能选择") + "\n\n" + m.menu.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		return m.renderFrame(viewStr)
	case mainStepAction:
		viewStr = header + "\n" + panel.Render(m.renderActionView())
		return m.renderFrame(viewStr)
	case mainStepOutput:
		if m.err != nil {
			viewStr = header + "\n" + panel.Render(m.titleStyle.Render("执行完成（含错误）")+"\n\n"+m.output+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单"))
			return m.renderFrame(viewStr)
		}
		viewStr = header + "\n" + panel.Render(m.titleStyle.Render("执行完成")+"\n\n"+m.output+"\n\n"+m.hintStyle.Render("按 Enter 返回菜单"))
		return m.renderFrame(viewStr)
	default:
		if m.running {
			spin := m.sp.View()
			viewStr = header + "\n" + panel.Render(m.titleStyle.Render("执行中")+"\n\n"+spin+" 正在等待 API 返回…")
			return m.renderFrame(viewStr)
		}
		viewStr = header
		return m.renderFrame(viewStr)
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
						if !meta.SupportedBaseModels[it.title] {
							m.err = fmt.Errorf("不支持的 Base Model: %s", it.title)
							return nil
						}
						m.act.cur.base = it.title
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
			// 路径输入框 + 文件选择器
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
				// 实时同步：当输入为有效目录时，自动刷新下方文件选择器
				typedPath := strings.TrimSpace(m.inpPath.Value())
				if typedPath != "" {
					if info, err := os.Stat(typedPath); err == nil && info.IsDir() {
						// 仅在目录变化时刷新，避免无谓刷新
						if filepath.Clean(m.filepicker.CurrentDirectory) != filepath.Clean(typedPath) {
							m.filepicker.CurrentDirectory = typedPath
							fpCmd = m.filepicker.Init()
						}
					}
				}
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
						// 确认是文件才进行完整的格式验证（包括扩展名）
						if err := m.validateAndSetPath(path); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
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
		// 列表与详情的双态
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc", "q":
				// 优先取消删除确认
				if m.confirmingDelete {
					m.confirmingDelete = false
					m.deleteTargetId = 0
					m.deleteTargetName = ""
					return nil
				}
				if m.viewingModelDetail {
					// 从详情返回列表
					m.viewingModelDetail = false
					m.modelDetail = nil
					return nil
				}
				// 返回菜单
				m.step = mainStepMenu
				m.act = actionInputs{}
				m.loadingModelList = false
				m.modelList = nil
				m.viewingModelDetail = false
				m.modelDetail = nil
				return nil
			case "r", "ctrl+r":
				if m.viewingModelDetail {
					// 详情页触发删除确认
					if m.modelDetail != nil {
						m.confirmingDelete = true
						m.deleteTargetId = m.modelDetail.Id
						m.deleteTargetName = m.modelDetail.Name
					}
					return nil
				}
				// 列表态刷新
				m.loadingModelList = true
				return loadModelList(m.apiKey)
			case "ctrl+d":
				// 仅在列表态允许删除
				if !m.viewingModelDetail && !m.loadingModelList && len(m.modelList) > 0 {
					selectedRow := m.modelTable.SelectedRow()
					if len(selectedRow) > 0 {
						idStr := selectedRow[0]
						nameStr := ""
						if len(selectedRow) > 1 {
							nameStr = selectedRow[1]
						}
						if idStr != "" {
							if id64, err := strconv.ParseInt(idStr, 10, 64); err == nil {
								m.confirmingDelete = true
								m.deleteTargetId = id64
								m.deleteTargetName = nameStr
								return nil
							}
						}
					}
				}
			case "enter":
				// 在删除确认界面按 Enter 确认删除
				if m.confirmingDelete && m.deleteTargetId > 0 {
					m.running = true
					return deleteBizyModel(m.apiKey, m.deleteTargetId)
				}
			}
		}
		if m.viewingModelDetail {
			// 详情页不需要 table 更新
			return nil
		}
		if m.confirmingDelete {
			// 确认界面不更新表格
			return nil
		}
		// 列表态：更新 table
		var cmd tea.Cmd
		m.modelTable, cmd = m.modelTable.Update(msg)
		return cmd

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
			if _, ih := m.innerSize(); ih > 0 {
				h := ih - 10
				if h < 5 {
					h = 5
				}
				m.typeList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 2/8 · 选择模型类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepVersion:
			return m.titleStyle.Render("上传 · Step 3/8 · 版本名称（默认 v1.0）") + "\n\n" + m.inpVersion.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
		case stepBase:
			if _, ih := m.innerSize(); ih > 0 {
				h := ih - 10
				if h < 5 {
					h = 5
				}
				m.baseList.SetHeight(h)
			}
			return m.titleStyle.Render("上传 · Step 4/8 · Base Model（必选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("选择后 Enter，返回：Esc，退出：q")
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
				pathInputLabel = m.titleStyle.Render("► 路径输入：（当前焦点，按Tab切换至文件选择器）")
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
				filePickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按Tab切换至路径输入框）")
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
				content.WriteString(m.hintStyle.Render("输入有效目录将自动同步下方文件列表；Enter确认文件或切换目录；Tab切换焦点；Esc返回"))
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
					content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Tab切换输入（输入框实时同步），Esc返回"))
				} else {
					content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Tab切换输入（输入框实时同步），Esc返回"))
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
			// 动态设置 moreList 高度
			if _, ih := m.innerSize(); ih > 0 {
				h := ih - 12
				if h < 5 {
					h = 5
				}
				m.moreList.SetHeight(h)
			}
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
		// 删除确认视图（优先级最高）
		if m.confirmingDelete {
			var b strings.Builder
			title := "删除确认"
			if m.deleteTargetName != "" {
				title = title + " · " + m.deleteTargetName
			}
			b.WriteString(m.titleStyle.Render(title))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("确定删除模型 #%d 吗？此操作不可恢复。\n\n", m.deleteTargetId))
			b.WriteString(m.hintStyle.Render("确认：Enter，取消：Esc"))
			return b.String()
		}

		if m.viewingModelDetail {
			// 详情视图
			if m.loadingModelDetail {
				return m.titleStyle.Render("模型详情") + "\n\n" + m.sp.View() + " 加载中...\n\n" + m.hintStyle.Render("返回：Esc/q")
			}
			if m.modelDetail == nil {
				return m.titleStyle.Render("模型详情") + "\n\n" + "未加载到数据" + "\n\n" + m.hintStyle.Render("返回：Esc/q")
			}
			// 渲染详情（按版本分块）
			var b strings.Builder
			b.WriteString(m.titleStyle.Render(fmt.Sprintf("%s (#%d) · %s", m.modelDetail.Name, m.modelDetail.Id, m.modelDetail.Type)))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("作者：%s\n创建：%s  更新：%s\n", dash(m.modelDetail.UserName), dash(m.modelDetail.CreatedAt), dash(m.modelDetail.UpdatedAt)))
			b.WriteString("\n")
			if len(m.modelDetail.Versions) == 0 {
				b.WriteString("暂无版本\n")
			} else {
				for i, v := range m.modelDetail.Versions {
					b.WriteString(m.hintStyle.Render(fmt.Sprintf("[%d] 版本 %s", i+1, dash(v.Version))))
					b.WriteString("\n")
					b.WriteString(fmt.Sprintf("  BaseModel: %s\n", dash(v.BaseModel)))
					if v.FileSize > 0 {
						b.WriteString(fmt.Sprintf("  文件: %s (%s)\n", dash(v.FileName), formatBytes(v.FileSize)))
					} else {
						b.WriteString(fmt.Sprintf("  文件: %s\n", dash(v.FileName)))
					}
					if v.Intro != "" {
						b.WriteString(fmt.Sprintf("  介绍: %s\n", v.Intro))
					}
					// 显示 model_id
					if v.ModelId > 0 {
						b.WriteString(fmt.Sprintf("  model_id: %d\n", v.ModelId))
					}
					b.WriteString(fmt.Sprintf("  创建: %s  更新: %s\n", dash(v.CreatedAt), dash(v.UpdatedAt)))
					b.WriteString("\n")
				}
			}
			b.WriteString(m.hintStyle.Render("返回：Esc/q，删除：Ctrl+R，列表：Enter 选择前需回退到列表"))
			return b.String()
		}

		// 列表视图
		if m.loadingModelList {
			return m.titleStyle.Render("列出模型") + "\n\n" +
				m.sp.View() + " 加载中...\n\n" +
				m.hintStyle.Render("请稍候")
		}
		if len(m.modelList) == 0 {
			return m.titleStyle.Render("列出模型") + "\n\n" +
				"暂无模型数据\n\n" +
				m.hintStyle.Render("返回：Esc/q，刷新：r")
		}
		// 调整 table 高度
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.modelTable.SetHeight(h)
		}
		// 确保列宽在渲染前与当前可用宽度一致
		if w, _ := m.innerSize(); w > 0 {
			m.resizeModelTable(w)
		}
		return m.titleStyle.Render(fmt.Sprintf("列出模型（共 %d 个）", m.modelListTotal)) + "\n\n" +
			m.modelTable.View() + "\n\n" +
			m.hintStyle.Render("导航：↑↓，进入详情：Enter，删除：Ctrl+D，返回：Esc/q，刷新：r")
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

			// 并行上传各版本文件，限制并发度
			mvList := make([]*lib.ModelVersion, len(versions))
			var wg sync.WaitGroup
			sem := make(chan struct{}, 3) // 默认并发 3，可调整
			var mu sync.Mutex
			var anyCanceled bool
			var errs []error

			for i, v := range versions {
				wg.Add(1)
				idx := i
				ver := v
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					// 统计与校验文件
					// 校验版本号，避免后端将空版本映射为 v0
					verVersion := strings.TrimSpace(ver.version)
					if verVersion == "" {
						mu.Lock()
						errs = append(errs, fmt.Errorf("版本[%d] 的版本号为空，请检查输入", idx+1))
						mu.Unlock()
						return
					}

					st, err := os.Stat(ver.path)
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}
					if st.IsDir() {
						mu.Lock()
						errs = append(errs, fmt.Errorf("仅支持文件上传: %s", ver.path))
						mu.Unlock()
						return
					}

					relPath, err := filepath.Rel(filepath.Dir(ver.path), ver.path)
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}
					f := &lib.FileToUpload{Path: filepath.ToSlash(ver.path), RelPath: filepath.ToSlash(relPath), Size: st.Size()}

					sha256sum, md5Hash, err := calculateHash(f.Path)
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}
					f.Signature = sha256sum

					ossCert, err := client.OssSign(sha256sum, u.typ)
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}
					fileIndex := fmt.Sprintf("%d/%d", idx+1, len(versions))
					fileRecord := ossCert.Data.File

					if fileRecord.Id == 0 {
						storage := ossCert.Data.Storage
						ossClient, err := lib.NewAliOssStorageClient(storage.Endpoint, storage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
						if err != nil {
							mu.Lock()
							errs = append(errs, err)
							mu.Unlock()
							return
						}
						_, err = ossClient.UploadFileCtx(ctx, f, fileRecord.ObjectKey, fileIndex, func(consumed, total int64) {
							select {
							case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: consumed, total: total, verIdx: idx}:
							default:
							}
						})
						if err != nil {
							if errors.Is(err, context.Canceled) {
								mu.Lock()
								anyCanceled = true
								mu.Unlock()
								return
							}
							mu.Lock()
							errs = append(errs, err)
							mu.Unlock()
							return
						}
						if _, err = client.CommitFileV2(f.Signature, fileRecord.ObjectKey, md5Hash, u.typ); err != nil {
							mu.Lock()
							errs = append(errs, err)
							mu.Unlock()
							return
						}
					} else {
						f.Id = fileRecord.Id
						f.RemoteKey = fileRecord.ObjectKey
						select {
						case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: f.Size, total: f.Size, verIdx: idx}:
						default:
						}
					}

					// 组装版本（成功的才入列）
					mv := &lib.ModelVersion{Version: verVersion, BaseModel: ver.base, Introduction: ver.intro, Public: false, Sign: f.Signature, Path: ver.path}
					if ver.cover != "" {
						mv.CoverUrls = strings.Split(ver.cover, ";")
					}
					mvList[idx] = mv
				}()
			}

			wg.Wait()

			// 若被取消，直接返回已取消
			if anyCanceled {
				ch <- actionDoneMsg{out: "上传已取消\n", err: nil}
				return
			}

			// 过滤成功的版本
			finalVersions := make([]*lib.ModelVersion, 0, len(mvList))
			for _, mv := range mvList {
				if mv != nil {
					finalVersions = append(finalVersions, mv)
				}
			}

			if len(finalVersions) == 0 {
				// 全部失败
				var sb strings.Builder
				sb.WriteString("所有版本上传失败\n")
				for _, e := range errs {
					sb.WriteString("- ")
					sb.WriteString(e.Error())
					sb.WriteString("\n")
				}
				ch <- actionDoneMsg{out: sb.String(), err: fmt.Errorf("上传失败")}
				return
			}

			// 发布模型（仅包含成功的版本）
			if _, err := client.CommitModelV2(u.name, u.typ, finalVersions); err != nil {
				ch <- actionDoneMsg{err: err}
				return
			}

			var out strings.Builder
			out.WriteString("Uploaded successfully\n")
			if len(finalVersions) != len(versions) {
				out.WriteString(fmt.Sprintf("部分版本失败：成功 %d / %d\n", len(finalVersions), len(versions)))
				for _, e := range errs {
					out.WriteString("- ")
					out.WriteString(e.Error())
					out.WriteString("\n")
				}
			}
			ch <- actionDoneMsg{out: out.String(), err: nil}
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

// 截断字符串辅助函数
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// 字符串数组去重
func uniqueStrings(strings []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, str := range strings {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

// 加载模型列表（直接调用API）
func loadModelList(apiKey string) tea.Cmd {
	return func() tea.Msg {
		// 创建客户端（使用默认域名）
		client := lib.NewClient(meta.DefaultDomain, apiKey)

		// 构建所有模型类型
		var modelTypes []string
		for _, t := range meta.ModelTypes {
			modelTypes = append(modelTypes, string(t))
		}

		// 调用 API（current=1, pageSize=100, keyword="", sort="Recently"）
		resp, err := client.ListModel(1, 100, "", "Recently", modelTypes, []string{})
		if err != nil {
			return modelListLoadedMsg{err: err}
		}

		return modelListLoadedMsg{
			models: resp.Data.List,
			total:  resp.Data.Total,
			err:    nil,
		}
	}
}

// 运行列出模型（含 public 开关）- 保留用于命令行模式
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
