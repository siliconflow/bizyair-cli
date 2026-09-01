package cmd

import (
	"os"
	"runtime"

	"github.com/siliconflow/bizyair-cli/meta"
)

// stdoutIsTTY 报告 stdout 是否连接到终端（而非管道/重定向）。
func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// colorEnabled 报告是否应该输出 ANSI 颜色：
// 仅当 stdout 是终端且运行在支持 ANSI 的操作系统上时启用。
func colorEnabled() bool {
	return stdoutIsTTY() && runtime.GOOS != meta.OSWindows
}

const ansiReset = "\033[0m"

func ansi(s, code string) string {
	if !colorEnabled() {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string   { return ansi(s, "\033[1m") }
func dim(s string) string    { return ansi(s, "\033[2m") }
func gray(s string) string   { return ansi(s, "\033[90m") }
func green(s string) string  { return ansi(s, "\033[32m") }
func red(s string) string    { return ansi(s, "\033[31m") }
func yellow(s string) string { return ansi(s, "\033[33m") }
func blue(s string) string   { return ansi(s, "\033[34m") }
func cyan(s string) string   { return ansi(s, "\033[36m") }

// clearLine 清空当前行并把光标移回行首（仅终端模式有效）。
func clearLine() {
	if stdoutIsTTY() {
		os.Stdout.WriteString("\r\033[2K")
	}
}