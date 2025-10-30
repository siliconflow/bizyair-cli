package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Manifest 发布清单文件结构
type Manifest struct {
	LatestVersion string              `json:"latest_version"`
	Releases      map[string]*Release `json:"releases"`
}

// Release 单个版本的发布信息
type Release struct {
	Version     string                     `json:"version"`
	ReleaseDate string                     `json:"release_date"`
	Platforms   map[string]*PlatformBinary `json:"platforms"`
}

// PlatformBinary 单个平台的二进制文件信息
type PlatformBinary struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"` // SHA256
}

// LoadManifestFromURL 从 URL 加载 manifest.json
func LoadManifestFromURL(url string) (*Manifest, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	return &manifest, nil
}

// LoadManifestFromFile 从本地文件加载 manifest.json
func LoadManifestFromFile(path string) (*Manifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	return &manifest, nil
}

// SaveManifestToFile 保存 manifest 到文件
func SaveManifestToFile(manifest *Manifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}

	return nil
}

// GetPlatformKey 根据 GOOS 和 GOARCH 生成平台标识符
func GetPlatformKey(goos, goarch string) string {
	// 将 darwin 转换为 macos
	if goos == "darwin" {
		goos = "macos"
	}
	return fmt.Sprintf("%s-%s", goos, goarch)
}

// GetBinaryForPlatform 获取指定平台的二进制文件信息
func (m *Manifest) GetBinaryForPlatform(goos, goarch string) (*PlatformBinary, error) {
	if m.LatestVersion == "" {
		return nil, fmt.Errorf("no latest version found in manifest")
	}

	release, ok := m.Releases[m.LatestVersion]
	if !ok {
		return nil, fmt.Errorf("release %s not found in manifest", m.LatestVersion)
	}

	platformKey := GetPlatformKey(goos, goarch)
	binary, ok := release.Platforms[platformKey]
	if !ok {
		return nil, fmt.Errorf("platform %s not found in release %s", platformKey, m.LatestVersion)
	}

	return binary, nil
}
