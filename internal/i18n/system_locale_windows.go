//go:build windows

package i18n

import "golang.org/x/sys/windows"

var getUserDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

func systemLocale() string {
	const (
		primaryLanguageMask = 0x3ff
		chineseLanguage     = 0x04
	)
	languageID, _, _ := getUserDefaultUILanguage.Call()
	if languageID&primaryLanguageMask == chineseLanguage {
		return "zh"
	}
	return ""
}
