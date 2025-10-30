// Package filepicker provides a file picker component for Bubble Tea
// applications.
package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

// New returns a new filepicker model with default styling and key bindings.
func New() Model {
	return Model{
		id:               nextID(),
		CurrentDirectory: ".",
		Cursor:           ">",
		AllowedTypes:     []string{},
		selected:         0,
		ShowPermissions:  true,
		ShowSize:         true,
		ShowHidden:       false,
		DirAllowed:       false,
		FileAllowed:      true,
		AutoHeight:       true,
		Height:           0,
		max:              0,
		min:              0,
		selectedStack:    newStack(),
		minStack:         newStack(),
		maxStack:         newStack(),
		KeyMap:           DefaultKeyMap(),
		Styles:           DefaultStyles(),
	}
}

type errorMsg struct {
	err error
}

type readDirMsg struct {
	id      int
	entries []os.DirEntry
}

const (
	marginBottom  = 5
	fileSizeWidth = 7
	paddingLeft   = 2
)

// KeyMap defines key bindings for each user action.
type KeyMap struct {
	GoToTop  key.Binding
	GoToLast key.Binding
	Down     key.Binding
	Up       key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Back     key.Binding
	Open     key.Binding
	Select   key.Binding
}

// DefaultKeyMap defines the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),
		Down:     key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Up:       key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		PageUp:   key.NewBinding(key.WithKeys("K", "pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("J", "pgdown"), key.WithHelp("pgdown", "page down")),
		Back:     key.NewBinding(key.WithKeys("h", "backspace", "left", "esc"), key.WithHelp("h", "back")),
		Open:     key.NewBinding(key.WithKeys("l", "right", "enter"), key.WithHelp("l", "open")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}
}

// Styles defines the possible customizations for styles in the file picker.
type Styles struct {
	DisabledCursor   lipgloss.Style
	Cursor           lipgloss.Style
	Symlink          lipgloss.Style
	Directory        lipgloss.Style
	File             lipgloss.Style
	DisabledFile     lipgloss.Style
	Permission       lipgloss.Style
	Selected         lipgloss.Style
	DisabledSelected lipgloss.Style
	FileSize         lipgloss.Style
	EmptyDirectory   lipgloss.Style
}

// DefaultStyles defines the default styling for the file picker.
func DefaultStyles() Styles {
	return DefaultStylesWithRenderer(lipgloss.DefaultRenderer())
}

// DefaultStylesWithRenderer defines the default styling for the file picker,
// with a given Lip Gloss renderer.
func DefaultStylesWithRenderer(r *lipgloss.Renderer) Styles {
	return Styles{
		DisabledCursor:   r.NewStyle().Foreground(lipgloss.Color("247")),
		Cursor:           r.NewStyle().Foreground(lipgloss.Color("212")),
		Symlink:          r.NewStyle().Foreground(lipgloss.Color("36")),
		Directory:        r.NewStyle().Foreground(lipgloss.Color("99")),
		File:             r.NewStyle(),
		DisabledFile:     r.NewStyle().Foreground(lipgloss.Color("243")),
		DisabledSelected: r.NewStyle().Foreground(lipgloss.Color("247")),
		Permission:       r.NewStyle().Foreground(lipgloss.Color("244")),
		Selected:         r.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
		FileSize:         r.NewStyle().Foreground(lipgloss.Color("240")).Width(fileSizeWidth).Align(lipgloss.Right),
		EmptyDirectory:   r.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(paddingLeft).SetString("Bummer. No Files Found."),
	}
}

// Model represents a file picker.
type Model struct {
	id int

	// Path is the path which the user has selected with the file picker.
	Path string

	// CurrentDirectory is the directory that the user is currently in.
	CurrentDirectory string

	// AllowedTypes specifies which file types the user may select.
	// If empty the user may select any file.
	AllowedTypes []string

	// FilterPrefix 文件名过滤前缀（用于实时过滤显示）
	FilterPrefix string

	KeyMap          KeyMap
	files           []os.DirEntry
	ShowPermissions bool
	ShowSize        bool
	ShowHidden      bool
	DirAllowed      bool
	FileAllowed     bool

	FileSelected  string
	selected      int
	selectedStack stack

	min      int
	max      int
	maxStack stack
	minStack stack

	// Height of the picker.
	//
	// Deprecated: use [Model.SetHeight] instead.
	Height     int
	AutoHeight bool

	Cursor string
	Styles Styles
}

type stack struct {
	Push   func(int)
	Pop    func() int
	Length func() int
}

func newStack() stack {
	slice := make([]int, 0)
	return stack{
		Push: func(i int) {
			slice = append(slice, i)
		},
		Pop: func() int {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Length: func() int {
			return len(slice)
		},
	}
}

func (m *Model) pushView(selected, minimum, maximum int) {
	m.selectedStack.Push(selected)
	m.minStack.Push(minimum)
	m.maxStack.Push(maximum)
}

func (m *Model) popView() (int, int, int) {
	return m.selectedStack.Pop(), m.minStack.Pop(), m.maxStack.Pop()
}

func (m Model) readDir(path string, showHidden bool) tea.Cmd {
	return func() tea.Msg {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return errorMsg{err}
		}

		sort.Slice(dirEntries, func(i, j int) bool {
			if dirEntries[i].IsDir() == dirEntries[j].IsDir() {
				return dirEntries[i].Name() < dirEntries[j].Name()
			}
			return dirEntries[i].IsDir()
		})

		var filteredEntries []os.DirEntry
		for _, dirEntry := range dirEntries {
			// 处理隐藏文件
			if !showHidden {
				isHidden, _ := IsHidden(dirEntry.Name())
				if isHidden {
					continue
				}
			}

			// 目录始终显示（用于导航）
			if dirEntry.IsDir() {
				filteredEntries = append(filteredEntries, dirEntry)
				continue
			}

			// 文件类型过滤
			if len(m.AllowedTypes) > 0 {
				// 只添加符合 AllowedTypes 的文件
				if m.canSelect(dirEntry.Name()) {
					filteredEntries = append(filteredEntries, dirEntry)
				}
			} else {
				// 没有设置 AllowedTypes，显示所有文件
				filteredEntries = append(filteredEntries, dirEntry)
			}
		}

		return readDirMsg{id: m.id, entries: filteredEntries}
	}
}

// Init initializes the file picker model.
func (m Model) Init() tea.Cmd {
	return m.readDir(m.CurrentDirectory, m.ShowHidden)
}

// SetHeight sets the height of the filepicker.
func (m *Model) SetHeight(height int) {
	m.Height = height
	if m.max > m.Height-1 {
		m.max = m.min + m.Height - 1
	}
}

// Update handles user interactions within the file picker model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case readDirMsg:
		if msg.id != m.id {
			break
		}
		m.files = msg.entries
		m.max = max(m.max, m.Height-1)

		// 修复：当文件列表变化后，确保 selected 索引有效
		if len(m.files) == 0 {
			m.selected = 0
		} else if m.selected >= len(m.files) {
			m.selected = len(m.files) - 1
		}

		// 如果当前选中的文件不可见（被过滤隐藏），跳转到第一个可见的文件
		if len(m.files) > 0 && !m.isFileVisible(m.selected) {
			m.selected = m.findFirstVisibleFile()
		}

		// 同时调整 min 和 max 以确保视图范围有效
		if m.min >= len(m.files) && len(m.files) > 0 {
			m.min = 0
		}
		if m.max >= len(m.files) && len(m.files) > 0 {
			m.max = len(m.files) - 1
		}
	case tea.WindowSizeMsg:
		if m.AutoHeight {
			m.Height = msg.Height - marginBottom
		}
		m.max = m.Height - 1
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.GoToTop):
			m.selected = m.findFirstVisibleFile()
			m.min = 0
			m.max = m.Height - 1
		case key.Matches(msg, m.KeyMap.GoToLast):
			m.selected = m.findLastVisibleFile()
			m.min = len(m.files) - m.Height
			m.max = len(m.files) - 1
		case key.Matches(msg, m.KeyMap.Down):
			// 查找下一个可见的文件
			nextVisible := m.findNextVisibleFile(m.selected + 1)
			if nextVisible != m.selected+1 || m.isFileVisible(nextVisible) {
				m.selected = nextVisible
			}
			if m.selected >= len(m.files) {
				m.selected = len(m.files) - 1
			}
			if m.selected > m.max {
				m.min++
				m.max++
			}
		case key.Matches(msg, m.KeyMap.Up):
			// 查找上一个可见的文件
			prevVisible := m.findPrevVisibleFile(m.selected - 1)
			if prevVisible != m.selected-1 || m.isFileVisible(prevVisible) {
				m.selected = prevVisible
			}
			if m.selected < 0 {
				m.selected = 0
			}
			if m.selected < m.min {
				m.min--
				m.max--
			}
		case key.Matches(msg, m.KeyMap.PageDown):
			m.selected += m.Height
			if m.selected >= len(m.files) {
				m.selected = len(m.files) - 1
			}
			m.min += m.Height
			m.max += m.Height

			if m.max >= len(m.files) {
				m.max = len(m.files) - 1
				m.min = m.max - m.Height
			}
		case key.Matches(msg, m.KeyMap.PageUp):
			m.selected -= m.Height
			if m.selected < 0 {
				m.selected = 0
			}
			m.min -= m.Height
			m.max -= m.Height

			if m.min < 0 {
				m.min = 0
				m.max = m.min + m.Height
			}
		case key.Matches(msg, m.KeyMap.Back):
			m.CurrentDirectory = filepath.Dir(m.CurrentDirectory)
			if m.selectedStack.Length() > 0 {
				m.selected, m.min, m.max = m.popView()
			} else {
				m.selected = 0
				m.min = 0
				m.max = m.Height - 1
			}
			return m, m.readDir(m.CurrentDirectory, m.ShowHidden)
		case key.Matches(msg, m.KeyMap.Open):
			if len(m.files) == 0 {
				break
			}

			f := m.files[m.selected]
			info, err := f.Info()
			if err != nil {
				break
			}
			isSymlink := info.Mode()&os.ModeSymlink != 0
			isDir := f.IsDir()

			if isSymlink {
				symlinkPath, _ := filepath.EvalSymlinks(filepath.Join(m.CurrentDirectory, f.Name()))
				info, err := os.Stat(symlinkPath)
				if err != nil {
					break
				}
				if info.IsDir() {
					isDir = true
				}
			}

			if (!isDir && m.FileAllowed) || (isDir && m.DirAllowed) {
				if key.Matches(msg, m.KeyMap.Select) {
					// Select the current path as the selection
					m.Path = filepath.Join(m.CurrentDirectory, f.Name())
				}
			}

			if !isDir {
				break
			}

			m.CurrentDirectory = filepath.Join(m.CurrentDirectory, f.Name())
			m.pushView(m.selected, m.min, m.max)
			m.selected = 0
			m.min = 0
			m.max = m.Height - 1
			return m, m.readDir(m.CurrentDirectory, m.ShowHidden)
		}
	}
	return m, nil
}

// View returns the view of the file picker.
func (m Model) View() string {
	if len(m.files) == 0 {
		return m.Styles.EmptyDirectory.Height(m.Height).MaxHeight(m.Height).String()
	}
	var s strings.Builder

	for i, f := range m.files {
		if i < m.min || i > m.max {
			continue
		}

		name := f.Name()

		// 应用过滤：跳过不匹配的文件和目录
		if !m.matchesFilter(name) {
			continue
		}

		var symlinkPath string
		info, _ := f.Info()
		isSymlink := info.Mode()&os.ModeSymlink != 0
		size := strings.Replace(humanize.Bytes(uint64(info.Size())), " ", "", 1) //nolint:gosec

		if isSymlink {
			symlinkPath, _ = filepath.EvalSymlinks(filepath.Join(m.CurrentDirectory, name))
		}

		disabled := !m.canSelect(name) && !f.IsDir()

		if m.selected == i { //nolint:nestif
			selected := ""
			if m.ShowPermissions {
				selected += " " + info.Mode().String()
			}
			if m.ShowSize {
				selected += fmt.Sprintf("%"+strconv.Itoa(m.Styles.FileSize.GetWidth())+"s", size)
			}
			selected += " " + name
			if isSymlink {
				selected += " → " + symlinkPath
			}
			if disabled {
				s.WriteString(m.Styles.DisabledSelected.Render(m.Cursor) + m.Styles.DisabledSelected.Render(selected))
			} else {
				s.WriteString(m.Styles.Cursor.Render(m.Cursor) + m.Styles.Selected.Render(selected))
			}
			s.WriteRune('\n')
			continue
		}

		style := m.Styles.File
		if f.IsDir() {
			style = m.Styles.Directory
		} else if isSymlink {
			style = m.Styles.Symlink
		} else if disabled {
			style = m.Styles.DisabledFile
		}

		fileName := style.Render(name)
		s.WriteString(m.Styles.Cursor.Render(" "))
		if isSymlink {
			fileName += " → " + symlinkPath
		}
		if m.ShowPermissions {
			s.WriteString(" " + m.Styles.Permission.Render(info.Mode().String()))
		}
		if m.ShowSize {
			s.WriteString(m.Styles.FileSize.Render(size))
		}
		s.WriteString(" " + fileName)
		s.WriteRune('\n')
	}

	for i := lipgloss.Height(s.String()); i <= m.Height; i++ {
		s.WriteRune('\n')
	}

	return s.String()
}

// DidSelectFile returns whether a user has selected a file (on this msg).
func (m Model) DidSelectFile(msg tea.Msg) (bool, string) {
	didSelect, path := m.didSelectFile(msg)
	if didSelect && m.canSelect(path) {
		return true, path
	}
	return false, ""
}

// DidSelectDisabledFile returns whether a user tried to select a disabled file
// (on this msg). This is necessary only if you would like to warn the user that
// they tried to select a disabled file.
func (m Model) DidSelectDisabledFile(msg tea.Msg) (bool, string) {
	didSelect, path := m.didSelectFile(msg)
	if didSelect && !m.canSelect(path) {
		return true, path
	}
	return false, ""
}

func (m Model) didSelectFile(msg tea.Msg) (bool, string) {
	if len(m.files) == 0 {
		return false, ""
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If the msg does not match the Select keymap then this could not have been a selection.
		if !key.Matches(msg, m.KeyMap.Select) {
			return false, ""
		}

		// The key press was a selection, let's confirm whether the current file could
		// be selected or used for navigating deeper into the stack.
		f := m.files[m.selected]
		info, err := f.Info()
		if err != nil {
			return false, ""
		}
		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := f.IsDir()

		if isSymlink {
			symlinkPath, _ := filepath.EvalSymlinks(filepath.Join(m.CurrentDirectory, f.Name()))
			info, err := os.Stat(symlinkPath)
			if err != nil {
				break
			}
			if info.IsDir() {
				isDir = true
			}
		}

		if (!isDir && m.FileAllowed) || (isDir && m.DirAllowed) && m.Path != "" {
			return true, m.Path
		}

		// If the msg was not a KeyMsg, then the file could not have been selected this iteration.
		// Only a KeyMsg can select a file.
	default:
		return false, ""
	}
	return false, ""
}

func (m Model) canSelect(file string) bool {
	if len(m.AllowedTypes) <= 0 {
		return true
	}

	for _, ext := range m.AllowedTypes {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}
	return false
}

// matchesFilter 检查文件名是否匹配过滤前缀
func (m Model) matchesFilter(name string) bool {
	if m.FilterPrefix == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(m.FilterPrefix))
}

// isFileVisible 检查文件是否可见（匹配过滤条件）
func (m Model) isFileVisible(index int) bool {
	if index < 0 || index >= len(m.files) {
		return false
	}
	return m.matchesFilter(m.files[index].Name())
}

// findNextVisibleFile 查找下一个可见的文件索引
func (m Model) findNextVisibleFile(start int) int {
	for i := start; i < len(m.files); i++ {
		if m.isFileVisible(i) {
			return i
		}
	}
	return start // 如果没找到，返回起始位置
}

// findPrevVisibleFile 查找上一个可见的文件索引
func (m Model) findPrevVisibleFile(start int) int {
	for i := start; i >= 0; i-- {
		if m.isFileVisible(i) {
			return i
		}
	}
	return start // 如果没找到，返回起始位置
}

// findFirstVisibleFile 查找第一个可见的文件索引
func (m Model) findFirstVisibleFile() int {
	return m.findNextVisibleFile(0)
}

// findLastVisibleFile 查找最后一个可见的文件索引
func (m Model) findLastVisibleFile() int {
	for i := len(m.files) - 1; i >= 0; i-- {
		if m.isFileVisible(i) {
			return i
		}
	}
	return 0
}
