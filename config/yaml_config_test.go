package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractVersionNumber(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    int
	}{
		{"v1.0 format", "v1.0", 1},
		{"v2.5 format", "v2.5", 2},
		{"v10.0 format", "v10.0", 10},
		{"no v-prefix", "3.0", 3},
		{"v-only number", "v5", 5},
		{"just number", "7", 7},
		{"empty string", "", 0},
		{"non-numeric", "abc", 0},
		{"v without number", "v", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersionNumber(tt.version)
			if got != tt.want {
				t.Errorf("extractVersionNumber(%q) = %d, want %d", tt.version, got, tt.want)
			}
		})
	}
}

func TestAutoIncrementVersionNames(t *testing.T) {
	tests := []struct {
		name     string
		versions []YamlVersion
		want     []string // expected Name for each version
	}{
		{
			"all empty names",
			[]YamlVersion{{}, {}, {}},
			[]string{"v1.0", "v2.0", "v3.0"},
		},
		{
			"some named some empty",
			[]YamlVersion{{Name: "v2.0"}, {}, {}},
			[]string{"v2.0", "v3.0", "v4.0"},
		},
		{
			"all named",
			[]YamlVersion{{Name: "v1.0"}, {Name: "v2.0"}},
			[]string{"v1.0", "v2.0"},
		},
		{
			"single empty",
			[]YamlVersion{{}},
			[]string{"v1.0"},
		},
		{
			"high starting number",
			[]YamlVersion{{Name: "v10.0"}, {}},
			[]string{"v10.0", "v11.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoIncrementVersionNames(tt.versions)
			for i, wantName := range tt.want {
				if got[i].Name != wantName {
					t.Errorf("version[%d].Name = %q, want %q", i, got[i].Name, wantName)
				}
			}
		})
	}
}

func TestLoadYamlConfig(t *testing.T) {
	tmpDir := t.TempDir()

	validYaml := `models:
  - name: "test-model"
    type: "LoRA"
    versions:
      - base_model: "SDXL"
        model_path: "model.safetensors"
        cover_url: "https://example.com/cover.jpg"
        intro: "test intro"
`
	validFile := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validFile, []byte(validYaml), 0644); err != nil {
		t.Fatal(err)
	}

	invalidYaml := `models: [invalid yaml`
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidFile, []byte(invalidYaml), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		wantLen int // number of models
	}{
		{"valid yaml", validFile, false, 1},
		{"invalid yaml", invalidFile, true, 0},
		{"non-existent file", filepath.Join(tmpDir, "nope.yaml"), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadYamlConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadYamlConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got.Models) != tt.wantLen {
				t.Errorf("LoadYamlConfig() models count = %d, want %d", len(got.Models), tt.wantLen)
			}
		})
	}
}

func TestValidateYamlConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *YamlConfig
		wantErr bool
	}{
		{
			"empty models",
			&YamlConfig{Models: []YamlModel{}},
			true,
		},
		{
			"empty model name",
			&YamlConfig{Models: []YamlModel{{Name: "", Type: "LoRA", Versions: []YamlVersion{{ModelPath: "test", CoverUrl: "http://x.jpg", Intro: "hi"}}}}},
			true,
		},
		{
			"invalid model type",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "InvalidType", Versions: []YamlVersion{{ModelPath: "test", CoverUrl: "http://x.jpg", Intro: "hi"}}}}},
			true,
		},
		{
			"no versions",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{}}}},
			true,
		},
		{
			"missing model_path",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{{CoverUrl: "http://x.jpg", Intro: "hi"}}}}},
			true, // model_path is empty
		},
		{
			"both cover_path and cover_url",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{{ModelPath: "test", CoverPath: "/local.jpg", CoverUrl: "http://x.jpg", Intro: "hi"}}}}},
			true,
		},
		{
			"neither cover_path nor cover_url",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{{ModelPath: "test", Intro: "hi"}}}}},
			true,
		},
		{
			"both intro and intro_path",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{{ModelPath: "test", CoverUrl: "http://x.jpg", Intro: "hi", IntroPath: "/intro.txt"}}}}},
			true,
		},
		{
			"neither intro nor intro_path",
			&YamlConfig{Models: []YamlModel{{Name: "test", Type: "LoRA", Versions: []YamlVersion{{ModelPath: "test", CoverUrl: "http://x.jpg"}}}}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateYamlConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateYamlConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeModelPaths(t *testing.T) {
	config := &YamlConfig{
		Models: []YamlModel{
			{
				Name: "test",
				Type: "LoRA",
				Versions: []YamlVersion{
					{ModelPath: "models/file.safetensors", CoverPath: "covers/cover.jpg", IntroPath: "descs/intro.txt"},
					{ModelPath: "/absolute/path/file.safetensors", CoverUrl: "https://example.com/cover.jpg"},
				},
			},
		},
	}

	err := NormalizeModelPaths(config, "/home/user/project")
	if err != nil {
		t.Fatal(err)
	}

	// First version: relative paths should be converted to absolute
	v0 := config.Models[0].Versions[0]
	if !filepath.IsAbs(v0.ModelPath) {
		t.Errorf("ModelPath should be absolute, got %q", v0.ModelPath)
	}
	if !filepath.IsAbs(v0.CoverPath) {
		t.Errorf("CoverPath should be absolute, got %q", v0.CoverPath)
	}
	if !filepath.IsAbs(v0.IntroPath) {
		t.Errorf("IntroPath should be absolute, got %q", v0.IntroPath)
	}

	// Second version: already absolute or URL should remain unchanged
	v1 := config.Models[0].Versions[1]
	if v1.ModelPath != "/absolute/path/file.safetensors" {
		t.Errorf("absolute ModelPath should be unchanged, got %q", v1.ModelPath)
	}
	if v1.CoverUrl != "https://example.com/cover.jpg" {
		t.Errorf("URL CoverUrl should be unchanged, got %q", v1.CoverUrl)
	}
}

func TestYamlVersion_GetCoverInput(t *testing.T) {
	tests := []struct {
		name  string
		v     YamlVersion
		want  string
	}{
		{"cover_path takes priority", YamlVersion{CoverPath: "/local.jpg", CoverUrl: "http://x.jpg"}, "/local.jpg"},
		{"cover_url fallback", YamlVersion{CoverUrl: "http://x.jpg"}, "http://x.jpg"},
		{"empty both", YamlVersion{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.GetCoverInput()
			if got != tt.want {
				t.Errorf("GetCoverInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYamlVersion_GetPublic(t *testing.T) {
	truth := true
	fals := false

	tests := []struct {
		name string
		v    YamlVersion
		want bool
	}{
		{"explicit true", YamlVersion{Public: &truth}, true},
		{"explicit false", YamlVersion{Public: &fals}, false},
		{"nil defaults to false", YamlVersion{Public: nil}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.GetPublic()
			if got != tt.want {
				t.Errorf("GetPublic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"https://example.com/file.jpg", true},
		{"http://example.com/file.jpg", true},
		{"/local/path/file.jpg", false},
		{"relative/path/file.jpg", false},
		{"ftp://example.com/file.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isURL(tt.path)
			if got != tt.want {
				t.Errorf("isURL(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
