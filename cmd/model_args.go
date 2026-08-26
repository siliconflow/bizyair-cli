package cmd

import (
	"strings"

	"github.com/urfave/cli/v2"
)

// firstModelArg 返回 args 中第一个非 flag 位置参数。
// urfave/cli 的 flag 解析在遇到第一个位置参数后停止，位置参数之后的
// flag（如 -y、-t）会残留在 args 中；此函数跳过这些残留 flag。
func firstModelArg(args cli.Args) string {
	for _, a := range args.Slice() {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// tailYesFlag 检测位置参数之后是否残留 -y / --yes 确认 flag。
func tailYesFlag(args cli.Args) bool {
	for _, a := range args.Slice() {
		switch a {
		case "-y", "--yes":
			return true
		}
		if strings.HasPrefix(a, "--yes=") {
			return true
		}
	}
	return false
}

// tailFlagValue 从位置参数之后的残留 flag 中提取带值 flag 的值。
// 支持 "-t LoRA"、"--type LoRA" 与 "--type=LoRA" 三种形式。
func tailFlagValue(args cli.Args, names ...string) (string, bool) {
	slice := args.Slice()
	for i, a := range slice {
		for _, n := range names {
			switch {
			case a == "-"+n || a == "--"+n:
				if i+1 < len(slice) {
					return slice[i+1], true
				}
			case strings.HasPrefix(a, "--"+n+"="):
				return strings.TrimPrefix(a, "--"+n+"="), true
			}
		}
	}
	return "", false
}