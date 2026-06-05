package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateModelName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "my-model", false},
		{"empty name", "", true},
		{"single char", "a", false},
		{"unicode name", "动漫模型", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateModelType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Checkpoint", "Checkpoint", false},
		{"LoRA", "LoRA", false},
		{"VAE", "VAE", false},
		{"UNet", "UNet", false},
		{"Controlnet", "Controlnet", false},
		{"CLIP", "CLIP", false},
		{"Upscaler", "Upscaler", false},
		{"Detection", "Detection", false},
		{"Other", "Other", false},
		{"empty type", "", true},
		{"invalid type", "InvalidType", true},
		{"lowercase lora", "lora", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "model.safetensors")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"existing file", tmpFile, false},
		{"empty path", "", true},
		{"non-existent path", "/nonexistent/path/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBaseModel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Flux.1 D", "Flux.1 D", false},
		{"SDXL", "SDXL", false},
		{"SD 1.5", "SD 1.5", false},
		{"Other", "Other", false},
		{"empty", "", true},
		{"invalid", "GPT-4", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedModels := []string{"Flux.1 D", "SDXL", "SD 1.5", "Other"}
			err := ValidateBaseModel(tt.input, allowedModels)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBaseModel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCoverFile(t *testing.T) {
	tmpDir := t.TempDir()

	validJpg := filepath.Join(tmpDir, "cover.jpg")
	validPng := filepath.Join(tmpDir, "cover.png")
	validWebp := filepath.Join(tmpDir, "cover.webp")
	validGif := filepath.Join(tmpDir, "cover.gif")
	validMp4 := filepath.Join(tmpDir, "cover.mp4")
	unsupportedTxt := filepath.Join(tmpDir, "cover.txt")
	oversizedMp4 := filepath.Join(tmpDir, "big.mp4")

	for _, f := range []string{validJpg, validPng, validWebp, validGif, validMp4, unsupportedTxt} {
		if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	oversizedData := make([]byte, 101*1024*1024)
	if err := os.WriteFile(oversizedMp4, oversizedData, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid jpg", validJpg, false},
		{"valid png", validPng, false},
		{"valid webp", validWebp, false},
		{"valid gif", validGif, false},
		{"valid mp4", validMp4, false},
		{"unsupported txt", unsupportedTxt, true},
		{"oversized mp4", oversizedMp4, true},
		{"non-existent file", filepath.Join(tmpDir, "nope.jpg"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoverFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCoverFile(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestIsSupportedCoverFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"jpg", "https://example.com/cover.jpg", true},
		{"jpeg", "https://example.com/cover.jpeg", true},
		{"png", "https://example.com/cover.png", true},
		{"gif", "https://example.com/cover.gif", true},
		{"webp", "https://example.com/cover.webp", true},
		{"mp4", "https://example.com/cover.mp4", true},
		{"webm", "https://example.com/cover.webm", true},
		{"mov", "https://example.com/cover.mov", true},
		{"url with query", "https://example.com/cover.jpg?width=100", true},
		{"unsupported pdf", "https://example.com/cover.pdf", false},
		{"unsupported svg", "https://example.com/cover.svg", false},
		{"no extension", "https://example.com/cover", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSupportedCoverFormat(tt.input)
			if got != tt.want {
				t.Errorf("IsSupportedCoverFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateIntroFile(t *testing.T) {
	tmpDir := t.TempDir()

	validTxt := filepath.Join(tmpDir, "intro.txt")
	validMd := filepath.Join(tmpDir, "intro.md")
	unsupportedHtml := filepath.Join(tmpDir, "intro.html")
	subDir := filepath.Join(tmpDir, "subdir")

	for _, f := range []string{validTxt, validMd, unsupportedHtml} {
		if err := os.WriteFile(f, []byte("test intro"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid txt", validTxt, false},
		{"valid md", validMd, false},
		{"unsupported html", unsupportedHtml, true},
		{"empty path", "", true},
		{"non-existent file", filepath.Join(tmpDir, "nope.txt"), true},
		{"directory", subDir, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIntroFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIntroFile(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestReadIntroFile(t *testing.T) {
	tmpDir := t.TempDir()

	shortContent := "Hello world"
	longContent := strings.Repeat("a", 6000)

	shortFile := filepath.Join(tmpDir, "short.txt")
	longFile := filepath.Join(tmpDir, "long.txt")

	if err := os.WriteFile(shortFile, []byte(shortContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(longFile, []byte(longContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantLen int
		wantErr bool
	}{
		{"short file", shortFile, len(shortContent), false},
		{"long file truncated to 5000", longFile, 5000, false},
		{"non-existent file", filepath.Join(tmpDir, "nope.txt"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadIntroFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadIntroFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len([]rune(got)) != tt.wantLen {
				t.Errorf("ReadIntroFile() rune length = %d, want %d", len([]rune(got)), tt.wantLen)
			}
		})
	}
}
