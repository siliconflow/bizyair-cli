package i18n

import (
	"regexp"
	"strings"
)

// apiKeyRe matches any character that is not a lowercase letter, digit, or underscore.
var apiKeyRe = regexp.MustCompile(`[^a-z0-9_]+`)

// APITranslate translates an API-returned value into the current language.
// domain is a logical grouping such as "category", "manufacturer", "billing_unit", "series", "tag".
// value is the raw string returned by the API (e.g. "Text To Image").
//
// It builds the i18n key  api.<domain>.<normalized_value>  where the value is
// lowercased and every run of non-alphanumeric characters is collapsed into a
// single underscore (e.g. "Text To Image" → "text_to_image").
//
// If a translation exists for the key, it is returned; otherwise the original
// value is returned unchanged, making the system safe for incremental additions.
func APITranslate(domain, value string) string {
	if value == "" {
		return ""
	}
	key := "api." + domain + "." + normalizeAPIKey(value)
	translated := T(key, nil)
	// T() returns "[missing translation: ...]" when the key is absent.
	if strings.HasPrefix(translated, "[missing translation:") {
		return value
	}
	return translated
}

// normalizeAPIKey converts "Text To Image" → "text_to_image",
// "3D Generation" → "3d_generation", etc.
func normalizeAPIKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = apiKeyRe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	// collapse repeated underscores
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}
