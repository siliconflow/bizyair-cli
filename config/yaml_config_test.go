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

func TestNormalizeYamlVersionInputs(t *testing.T) {
	yamlDir := t.TempDir()
	absoluteModelPath := filepath.Join(t.TempDir(), "file.safetensors")
	public := true
	config := &YamlConfig{
		Models: []YamlModel{
			{
				Name: "test",
				Type: "LoRA",
				Versions: []YamlVersion{
					{
						ModelPath: "models/file.safetensors",
						CoverPath: "covers/cover.jpg",
						IntroPath: "descs/intro.txt",
						Public:    &public,
					},
					{
						ModelPath: absoluteModelPath,
						CoverUrl:  "https://example.com/cover.jpg",
					},
				},
			},
		},
	}

	if err := NormalizeModelPaths(config, yamlDir); err != nil {
		t.Fatal(err)
	}

	v0 := config.Models[0].Versions[0]
	if want := filepath.Join(yamlDir, "models/file.safetensors"); v0.ModelPath != want {
		t.Errorf("ModelPath = %q, want %q", v0.ModelPath, want)
	}
	wantCover := filepath.Join(yamlDir, "covers/cover.jpg")
	if v0.GetCoverInput() != wantCover {
		t.Errorf("cover input = %q, want %q", v0.GetCoverInput(), wantCover)
	}
	if want := filepath.Join(yamlDir, "descs/intro.txt"); v0.IntroPath != want {
		t.Errorf("IntroPath = %q, want %q", v0.IntroPath, want)
	}
	if !v0.GetPublic() {
		t.Error("explicit public value was not preserved")
	}

	v1 := config.Models[0].Versions[1]
	if v1.ModelPath != absoluteModelPath {
		t.Errorf("absolute ModelPath should be unchanged, got %q", v1.ModelPath)
	}
	if want := "https://example.com/cover.jpg"; v1.GetCoverInput() != want {
		t.Errorf("cover input = %q, want %q", v1.GetCoverInput(), want)
	}
	if v1.GetPublic() {
		t.Error("unset public value should default to false")
	}
}
