package format

import (
	"reflect"
	"strings"
)

// ExtractOutputURLs 递归提取 outputs 中的 http(s) 链接。
func ExtractOutputURLs(outputs any) []string {
	var urls []string
	extractURLsRecursive(reflect.ValueOf(outputs), &urls)
	return urls
}

func extractURLsRecursive(v reflect.Value, urls *[]string) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			*urls = append(*urls, s)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			extractURLsRecursive(v.MapIndex(key), urls)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			extractURLsRecursive(v.Index(i), urls)
		}
	case reflect.Interface:
		extractURLsRecursive(v.Elem(), urls)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			extractURLsRecursive(v.Field(i), urls)
		}
	}
}