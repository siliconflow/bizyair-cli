package lib

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

type SfFolder struct {
}

func NewSfFolder() *SfFolder {
	return &SfFolder{}
}

func (s *SfFolder) folderPath(filePath string) string {
	currentOS := runtime.GOOS
	// 判断是否为 Windows
	if currentOS == meta.OSWindows {
		homeDir := os.Getenv(meta.EnvUserProfile)
		return filepath.Join(homeDir, meta.SfFolder, filePath)
	}
	return filepath.Join(os.Getenv(meta.EnvHome), meta.SfFolder, filePath)
}

func (s *SfFolder) SaveKey(apikey string) error {
	err := os.MkdirAll(s.folderPath(""), 0700)
	if err != nil {
		return i18n.NewError("error.io.create_directory_failed", map[string]any{"Path": s.folderPath("")}, err)
	}

	if runtime.GOOS != meta.OSWindows {
		err = os.Chmod(s.folderPath(""), 0700)
		if err != nil {
			return i18n.NewError("error.io.permissions_failed", map[string]any{"Path": s.folderPath("")}, err)
		}
	}

	keyFilePath := s.folderPath(meta.SfApiKey)
	file, err := os.OpenFile(keyFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return i18n.NewError("error.auth.key_open_failed", map[string]any{"Path": keyFilePath}, err)
	}
	defer file.Close()
	if runtime.GOOS != meta.OSWindows {
		if err := file.Chmod(0600); err != nil {
			return i18n.NewError("error.io.permissions_failed", map[string]any{"Path": keyFilePath}, err)
		}
	}

	if _, err := file.WriteString(apikey); err != nil {
		return i18n.NewError("error.auth.key_save_failed", map[string]any{"Path": keyFilePath}, err)
	}

	return nil
}

func (s *SfFolder) RemoveKey() error {
	keyFilePath := s.folderPath(meta.SfApiKey)
	_, err := os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return i18n.NewError("error.auth.not_logged_in_simple", nil, err)
	}
	err = os.Remove(keyFilePath)
	if err != nil {
		return err
	}
	return nil
}

func (s *SfFolder) GetKey() (string, error) {
	keyFilePath := s.folderPath(meta.SfApiKey)
	_, err := os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return "", i18n.NewError("error.auth.not_logged_in_simple", nil, err)
	}
	if err != nil {
		return "", i18n.NewError("error.auth.key_load_failed", map[string]any{"Path": keyFilePath}, err)
	}
	if runtime.GOOS != meta.OSWindows {
		if err := os.Chmod(keyFilePath, 0600); err != nil {
			return "", i18n.NewError("error.io.permissions_failed", map[string]any{"Path": keyFilePath}, err)
		}
	}
	// 读取文件内容
	content, err := os.ReadFile(keyFilePath)
	if err != nil {
		return "", i18n.NewError("error.auth.key_load_failed", map[string]any{"Path": keyFilePath}, err)
	}

	return string(content), nil
}
