package i18n

import (
	"errors"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

var templateVariable = regexp.MustCompile(`{{\s*\.([A-Za-z_][A-Za-z0-9_]*)`)

func env(values map[string]string) LookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    Language
		wantErr bool
	}{
		{name: "default English", want: English},
		{name: "Simplified Chinese system language", env: map[string]string{"LANG": "zh_CN.UTF-8"}, want: SimplifiedChinese},
		{name: "Traditional Chinese system language", env: map[string]string{"LANG": "zh_TW.UTF-8"}, want: SimplifiedChinese},
		{name: "Traditional Chinese script locale", env: map[string]string{"LANG": "zh-Hant-HK"}, want: SimplifiedChinese},
		{name: "English locale", env: map[string]string{"LANG": "en_US.UTF-8"}, want: English},
		{name: "explicit English", args: []string{"--lang", "en"}, want: English},
		{name: "explicit Simplified Chinese", args: []string{"--lang=zh-CN"}, want: SimplifiedChinese},
		{name: "unrelated locale", env: map[string]string{"LANG": "fr_FR.UTF-8"}, want: English},
		{name: "invalid system locale", env: map[string]string{"LANG": "not_a_locale"}, want: English},
		{name: "C locale", env: map[string]string{"LANG": "C"}, want: English},
		{name: "POSIX locale", env: map[string]string{"LANG": "POSIX"}, want: English},
		{name: "empty high priority locale is skipped", env: map[string]string{"LC_ALL": "", "LC_MESSAGES": "zh_CN"}, want: SimplifiedChinese},
		{name: "LC messages wins", env: map[string]string{"LC_MESSAGES": "zh_CN", "LANG": "en_US"}, want: SimplifiedChinese},
		{name: "LC all wins", env: map[string]string{"LC_ALL": "en_US", "LC_MESSAGES": "zh_CN"}, want: English},
		{name: "flag wins over system language", args: []string{"--lang=en"}, env: map[string]string{"LANG": "zh_CN"}, want: English},
		{name: "split flag", args: []string{"--verbose", "--lang", "zh-CN", "upload"}, want: SimplifiedChinese},
		{name: "last flag wins", args: []string{"--lang=zh-CN", "--lang", "en"}, want: English},
		{name: "invalid explicit flag", args: []string{"--lang", "fr"}, wantErr: true},
		{name: "English alias is rejected", args: []string{"--lang", "en-GB"}, wantErr: true},
		{name: "Chinese base alias is rejected", args: []string{"--lang=zh"}, wantErr: true},
		{name: "Traditional Chinese override is rejected", args: []string{"--lang=zh-TW"}, wantErr: true},
		{name: "missing flag", args: []string{"--lang"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args, env(tt.env))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTranslations(t *testing.T) {
	for _, tt := range []struct {
		lang Language
		want string
	}{
		{English, "Hello, BizyAir!"},
		{SimplifiedChinese, "你好，BizyAir！"},
	} {
		if err := Configure(tt.lang); err != nil {
			t.Fatal(err)
		}
		if got := T("test.greeting", map[string]any{"Name": "BizyAir"}); got != tt.want {
			t.Fatalf("T() = %q, want %q", got, tt.want)
		}
	}

	if err := Configure(English); err != nil {
		t.Fatal(err)
	}
	if got := TN("test.files", 1, map[string]any{"Count": 1}); got != "1 file" {
		t.Fatalf("TN(one) = %q", got)
	}
	if got := TN("test.files", 2, map[string]any{"Count": 2}); got != "2 files" {
		t.Fatalf("TN(other) = %q", got)
	}
	if err := Configure(SimplifiedChinese); err != nil {
		t.Fatal(err)
	}
	if got := TN("test.files", 2, map[string]any{"Count": 2}); got != "2 个文件" {
		t.Fatalf("TN(zh-CN) = %q", got)
	}
}

func TestLocalizedErrorUnwrap(t *testing.T) {
	if err := Configure(English); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("disk full")
	err := NewError("test.greeting", map[string]any{"Name": "failure"}, cause)
	if !errors.Is(err, cause) {
		t.Fatal("localized error did not preserve its cause")
	}
	var localized *LocalizedError
	if !errors.As(err, &localized) {
		t.Fatal("localized error type was not preserved")
	}
	if !reflect.DeepEqual(localized.Data, map[string]any{"Name": "failure"}) {
		t.Fatalf("unexpected data: %#v", localized.Data)
	}
}

func TestCatalogsHaveMatchingIDsAndVariables(t *testing.T) {
	english := readCatalog(t, "locales/active.en.yaml")
	chinese := readCatalog(t, "locales/active.zh-CN.yaml")

	englishIDs := sortedKeys(english)
	chineseIDs := sortedKeys(chinese)
	if !reflect.DeepEqual(englishIDs, chineseIDs) {
		t.Fatalf("catalog IDs differ\nEnglish only: %v\nChinese only: %v", difference(englishIDs, chineseIDs), difference(chineseIDs, englishIDs))
	}

	for _, id := range englishIDs {
		enVars := variables(english[id])
		zhVars := variables(chinese[id])
		if !reflect.DeepEqual(enVars, zhVars) {
			t.Errorf("%s template variables differ: en=%v zh-CN=%v", id, enVars, zhVars)
		}
		data := make(map[string]any, len(enVars))
		for _, variable := range enVars {
			data[variable] = "value"
		}
		data["Count"] = 2
		for _, lang := range []Language{English, SimplifiedChinese} {
			if err := Configure(lang); err != nil {
				t.Fatal(err)
			}
			got := T(id, data)
			if strings.HasPrefix(got, "[missing translation:") {
				t.Errorf("%s does not resolve for %s", id, lang)
			}
		}
	}
}

func readCatalog(t *testing.T, name string) map[string]string {
	t.Helper()
	contents, err := localeFS.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(contents, &document); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	messages := make(map[string]string)
	if len(document.Content) != 1 {
		t.Fatalf("%s has an invalid YAML document", name)
	}
	walkCatalog(t, name, document.Content[0], nil, messages)
	return messages
}

func walkCatalog(t *testing.T, name string, node *yaml.Node, path []string, messages map[string]string) {
	t.Helper()
	if node.Kind == yaml.ScalarNode {
		messages[strings.Join(path, ".")] = node.Value
		return
	}
	if node.Kind != yaml.MappingNode {
		t.Fatalf("%s: %s must be a mapping or scalar", name, strings.Join(path, "."))
	}

	reserved := map[string]bool{"description": true, "zero": true, "one": true, "two": true, "few": true, "many": true, "other": true, "leftdelim": true, "rightdelim": true}
	isMessage := false
	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if seen[key] {
			t.Fatalf("%s: duplicate key %s under %s", name, key, strings.Join(path, "."))
		}
		seen[key] = true
		isMessage = isMessage || reserved[key]
	}
	if isMessage {
		var text strings.Builder
		for i := 1; i < len(node.Content); i += 2 {
			text.WriteString(node.Content[i].Value)
			text.WriteByte('\n')
		}
		messages[strings.Join(path, ".")] = text.String()
		return
	}
	for i := 0; i < len(node.Content); i += 2 {
		walkCatalog(t, name, node.Content[i+1], append(path, node.Content[i].Value), messages)
	}
}

func variables(message string) []string {
	seen := map[string]bool{}
	for _, match := range templateVariable.FindAllStringSubmatch(message, -1) {
		seen[match[1]] = true
	}
	return sortedKeys(seen)
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func difference(left, right []string) []string {
	rightSet := make(map[string]bool, len(right))
	for _, value := range right {
		rightSet[value] = true
	}
	var result []string
	for _, value := range left {
		if !rightSet[value] {
			result = append(result, value)
		}
	}
	return result
}
