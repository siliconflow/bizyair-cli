package lib

import (
	"strings"
	"testing"
)

func TestDeriveModelzooRulesMediaSkipsEnumConstraint(t *testing.T) {
	cases := []struct {
		name    string
		param   ModelzooInputParam
		raw     string
		wantErr bool
		match   string
	}{
		{
			name: "image with preset options accepts arbitrary user URL",
			param: ModelzooInputParam{
				VariableType: "image",
				FieldLabel:   "first frame",
				Required:     true,
				FieldOptions: &ModelzooFieldOptions{Values: []any{"preset1.png", "preset2.png"}},
			},
			raw: "https://storage.bizyair.ai/inputs/user-uploaded-image.png",
		},
		{
			name: "video with preset options accepts arbitrary user URL",
			param: ModelzooInputParam{
				VariableType: "video",
				FieldLabel:   "source video",
				FieldOptions: &ModelzooFieldOptions{Values: []any{"preset.mp4"}},
			},
			raw: "https://example.com/user/clip.mp4",
		},
		{
			name: "media value must still be an http(s) URL",
			param: ModelzooInputParam{
				VariableType: "image",
				FieldLabel:   "first frame",
				FieldOptions: &ModelzooFieldOptions{Values: []any{"preset1.png"}},
			},
			raw:     "preset1.png",
			wantErr: true,
			match:   "URL",
		},
		{
			name: "required media still requires a value",
			param: ModelzooInputParam{
				VariableType: "image",
				FieldLabel:   "first frame",
				Required:     true,
			},
			raw:     "",
			wantErr: true,
		},
		{
			name: "string enum still enforces the option list",
			param: ModelzooInputParam{
				VariableType: "string",
				FieldLabel:   "model",
				FieldOptions: &ModelzooFieldOptions{Values: []any{"opt-a", "opt-b"}},
			},
			raw:     "opt-c",
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rules := DeriveModelzooRules(c.param)
			err := ValidateModelzooParam(rules, c.raw)
			if c.wantErr {
				if err == nil {
					t.Fatalf("raw %q: expected error, got none", c.raw)
				}
				if c.match != "" && !strings.Contains(err.Error(), c.match) {
					t.Fatalf("raw %q: error %q does not contain %q", c.raw, err.Error(), c.match)
				}
				return
			}
			if err != nil {
				t.Fatalf("raw %q: unexpected error: %v", c.raw, err)
			}
		})
	}
}
