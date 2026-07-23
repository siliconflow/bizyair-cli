package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	yaml "go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
)

// Language is a canonical language supported by BizyAir CLI.
type Language string

const (
	English           Language = "en"
	SimplifiedChinese Language = "zh-CN"
)

// LookupEnv matches os.LookupEnv and keeps language resolution easy to test.
type LookupEnv func(string) (string, bool)

//go:embed locales/*.yaml
var localeFS embed.FS

var state struct {
	initOnce sync.Once
	initErr  error
	bundle   *goi18n.Bundle

	mu        sync.RWMutex
	language  Language
	localizer *goi18n.Localizer
	english   *goi18n.Localizer
}

func initBundle() error {
	state.initOnce.Do(func() {
		bundle := goi18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
		if _, err := bundle.LoadMessageFileFS(localeFS, "locales/active.en.yaml"); err != nil {
			state.initErr = fmt.Errorf("load English translations: %w", err)
			return
		}
		if _, err := bundle.LoadMessageFileFS(localeFS, "locales/active.zh-CN.yaml"); err != nil {
			state.initErr = fmt.Errorf("load Chinese translations: %w", err)
			return
		}
		state.bundle = bundle
		state.english = goi18n.NewLocalizer(bundle, string(English))
	})
	return state.initErr
}

// Configure selects the process language. It is intended to be called once,
// before commands and TUI models are created. The loaded bundle is immutable.
func Configure(lang Language) error {
	if lang != English && lang != SimplifiedChinese {
		return fmt.Errorf("unsupported language %q; supported languages: en, zh-CN", lang)
	}
	if err := initBundle(); err != nil {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.language = lang
	state.localizer = goi18n.NewLocalizer(state.bundle, string(lang), string(English))
	return nil
}

func ensureConfigured() {
	state.mu.RLock()
	configured := state.localizer != nil
	state.mu.RUnlock()
	if configured {
		return
	}
	_ = Configure(English)
}

// CurrentLanguage returns the canonical active language.
func CurrentLanguage() Language {
	ensureConfigured()
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.language
}

// T localizes a message using optional named template data.
func T(messageID string, templateData ...map[string]any) string {
	return localize(messageID, nil, firstData(templateData))
}

// TN localizes a plural message using CLDR plural rules.
func TN(messageID string, count any, templateData ...map[string]any) string {
	return localize(messageID, count, firstData(templateData))
}

func localize(messageID string, count any, data map[string]any) string {
	ensureConfigured()
	state.mu.RLock()
	localizer := state.localizer
	english := state.english
	state.mu.RUnlock()

	config := &goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
		PluralCount:  count,
	}
	if message, err := localizer.Localize(config); err == nil {
		return message
	}
	if message, err := english.Localize(config); err == nil {
		return message
	}
	return "[missing translation: " + messageID + "]"
}

func firstData(all []map[string]any) map[string]any {
	if len(all) == 0 {
		return nil
	}
	return all[0]
}

// Resolve applies the language precedence used by the CLI:
// --lang > system language > English.
func Resolve(args []string, lookupEnv LookupEnv) (Language, error) {
	language, _, err := ResolveArgs(args, lookupEnv)
	return language, err
}

// ResolveArgs resolves the interface language and removes every recognized
// --lang option before the remaining arguments are handed to urfave/cli. This
// keeps language selection in one parser while allowing the global option on
// either side of a subcommand.
func ResolveArgs(args []string, lookupEnv LookupEnv) (Language, []string, error) {
	value, found, remaining, err := extractLanguageFlag(args)
	if err != nil {
		return English, nil, err
	}
	if found {
		resolved, resolveErr := parseExplicitLanguage(value)
		if resolveErr != nil {
			return English, nil, resolveErr
		}
		return resolved, remaining, nil
	}

	resolved := English
	// These variables are implementation details of system-language detection
	// on Unix-like systems, not additional BizyAir configuration knobs.
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if lookupEnv == nil {
			break
		}
		if value, ok := lookupEnv(name); ok && strings.TrimSpace(value) != "" {
			resolved = detectSystemLocale(value)
			break
		}
	}
	return resolved, remaining, nil
}

func extractLanguageFlag(args []string) (value string, found bool, remaining []string, err error) {
	remaining = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			remaining = append(remaining, args[i:]...)
			break
		}
		if arg == "--lang" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", false, nil, fmt.Errorf("--lang requires a value (en or zh-CN)")
			}
			value, found = args[i+1], true
			i++
			continue
		}
		if strings.HasPrefix(arg, "--lang=") {
			value, found = strings.TrimPrefix(arg, "--lang="), true
			if strings.TrimSpace(value) == "" {
				return "", false, nil, fmt.Errorf("--lang requires a value (en or zh-CN)")
			}
			continue
		}
		remaining = append(remaining, arg)
	}
	return value, found, remaining, nil
}

// parseExplicitLanguage validates --lang. Unlike automatic system detection,
// the override only accepts the two documented canonical values.
func parseExplicitLanguage(value string) (Language, error) {
	switch value {
	case string(English):
		return English, nil
	case string(SimplifiedChinese):
		return SimplifiedChinese, nil
	default:
		return English, fmt.Errorf("unsupported language %q; supported languages: en, zh-CN", value)
	}
}

func detectSystemLocale(value string) Language {
	normalized := normalizeLocale(value)
	if normalized == "" || normalized == "c" || normalized == "posix" {
		return English
	}
	if normalized == "zh" || strings.HasPrefix(normalized, "zh-") {
		return SimplifiedChinese
	}
	return English
}

func normalizeLocale(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if i := strings.IndexByte(value, '.'); i >= 0 {
		value = value[:i]
	}
	if i := strings.IndexByte(value, '@'); i >= 0 {
		value = value[:i]
	}
	return strings.ReplaceAll(value, "_", "-")
}
