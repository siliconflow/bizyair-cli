package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/urfave/cli/v2"
)

func runCLI(t *testing.T, lang i18n.Language, args ...string) (string, error) {
	t.Helper()
	if err := i18n.Configure(lang); err != nil {
		t.Fatal(err)
	}
	app := Init()
	app.ExitErrHandler = func(*cli.Context, error) {}
	var output bytes.Buffer
	app.Writer = &output
	app.ErrWriter = &output
	err := app.Run(append([]string{"bizyair"}, args...))
	return output.String(), err
}

func TestLocalizedRootAndCommandHelp(t *testing.T) {
	english, err := runCLI(t, i18n.English, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"USAGE:", "COMMANDS:", "Upload model files to BizyAir", "--lang value"} {
		if !strings.Contains(english, want) {
			t.Errorf("English root help does not contain %q\n%s", want, english)
		}
	}

	chinese, err := runCLI(t, i18n.SimplifiedChinese, "upload", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"用法:", "选项:", "上传模型文件到 BizyAir", "指定模型名称"} {
		if !strings.Contains(chinese, want) {
			t.Errorf("Chinese upload help does not contain %q\n%s", want, chinese)
		}
	}
	if strings.Contains(chinese, "USAGE:") || strings.Contains(chinese, "OPTIONS:") {
		t.Fatalf("Chinese help contains an English heading:\n%s", chinese)
	}
}

func TestLocalizedCLIInputErrors(t *testing.T) {
	_, err := runCLI(t, i18n.English, "not-a-command")
	if err == nil || err.Error() != "Unknown command: not-a-command" {
		t.Fatalf("unexpected English command error: %v", err)
	}

	_, err = runCLI(t, i18n.SimplifiedChinese, "not-a-command")
	if err == nil || err.Error() != "未知命令：not-a-command" {
		t.Fatalf("unexpected Chinese command error: %v", err)
	}

	_, err = runCLI(t, i18n.SimplifiedChinese, "--not-a-flag")
	if err == nil || !strings.Contains(err.Error(), "未知参数") {
		t.Fatalf("unexpected Chinese usage error: %v", err)
	}

	_, err = runCLI(t, i18n.SimplifiedChinese, "model", "rm")
	if err == nil || !strings.Contains(err.Error(), "模型类型不能为空") {
		t.Fatalf("unexpected Chinese required input error: %v", err)
	}
}

func TestLocalizedHelpCommand(t *testing.T) {
	output, err := runCLI(t, i18n.SimplifiedChinese, "help", "model")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "与模型交互") || !strings.Contains(output, "命令:") {
		t.Fatalf("help command was not localized:\n%s", output)
	}
}

func TestLocalizedVersionAndUpgradeHelp(t *testing.T) {
	englishVersion, err := runCLI(t, i18n.English, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(englishVersion, "Version:") || !strings.Contains(englishVersion, "Built At:") {
		t.Fatalf("English version output was not localized:\n%s", englishVersion)
	}

	chineseVersion, err := runCLI(t, i18n.SimplifiedChinese, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(chineseVersion, "版本：") || !strings.Contains(chineseVersion, "构建时间：") {
		t.Fatalf("Chinese version output was not localized:\n%s", chineseVersion)
	}

	upgradeHelp, err := runCLI(t, i18n.SimplifiedChinese, "upgrade", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(upgradeHelp, "仅检查更新") || !strings.Contains(upgradeHelp, "强制升级") {
		t.Fatalf("Chinese upgrade help was not localized:\n%s", upgradeHelp)
	}
}
