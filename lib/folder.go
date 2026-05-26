package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

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
	err := os.MkdirAll(s.folderPath(""), 0660)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if runtime.GOOS != meta.OSWindows {
		err = os.Chmod(s.folderPath(""), 0770)
		if err != nil {
			return fmt.Errorf("failed to set directory permissions: %w", err)
		}
	}

	keyFilePath := s.folderPath(meta.SfApiKey)
	file, err := os.OpenFile(keyFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open apikey failed: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(apikey); err != nil {
		return fmt.Errorf("save apikey failed: %w", err)
	}

	return nil
}

func (s *SfFolder) RemoveKey() error {
	keyFilePath := s.folderPath(meta.SfApiKey)
	_, err := os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s", meta.NotLoggedIn)
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
		return "", fmt.Errorf("%s", meta.NotLoggedIn)
	}
	// 读取文件内容
	content, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to load apikey file")
	}

	return string(content), nil
}
