package config

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLocalizedExamplesHaveMatchingFields(t *testing.T) {
	chinese := readExampleShape(t, "../example.yaml")
	english := readExampleShape(t, "../example.en.yaml")
	if !reflect.DeepEqual(chinese, english) {
		t.Fatalf("example.yaml and example.en.yaml have different field structures\nzh-CN: %#v\nen: %#v", chinese, english)
	}
}

func readExampleShape(t *testing.T, path string) any {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := yaml.Unmarshal(contents, &value); err != nil {
		t.Fatal(err)
	}
	return yamlShape(value)
}

func yamlShape(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		shape := make(map[string]any, len(typed))
		for key, child := range typed {
			shape[key] = yamlShape(child)
		}
		return shape
	case []any:
		shape := make([]any, len(typed))
		for index, child := range typed {
			shape[index] = yamlShape(child)
		}
		return shape
	default:
		return "scalar"
	}
}
