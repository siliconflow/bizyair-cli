package lib

import (
	"fmt"
	"github.com/siliconflow/siliconcloud-cli/meta"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
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
		return cli.Exit(err, meta.LoadError)
	}

	if runtime.GOOS != meta.OSWindows {
		err = os.Chmod(s.folderPath(""), 0770)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to set directory permissions: %w", err), meta.LoadError)
		}
	}

	keyFilePath := s.folderPath(meta.SfApiKey)
	file, err := os.OpenFile(keyFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return cli.Exit(fmt.Errorf("open apikey failed: %w", err), meta.LoadError)
	}
	defer file.Close()

	if _, err := file.WriteString(apikey); err != nil {
		return cli.Exit(fmt.Errorf("save apikey failed: %w", err), meta.LoadError)
	}

	return nil
}

func (s *SfFolder) RemoveKey() error {
	keyFilePath := s.folderPath(meta.SfApiKey)
	_, err := os.Stat(keyFilePath)
	if os.IsNotExist(err) {
		return cli.Exit(meta.NotLoggedIn, meta.LoadError)
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
		return "", cli.Exit(meta.NotLoggedIn, meta.LoadError)
	}
	// 读取文件内容
	content, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		return "", cli.Exit(fmt.Errorf("failed to load apikey file"), meta.LoadError)
	}

	return string(content), nil
}
