package lib

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

// UpgradeOptions 升级选项
type UpgradeOptions struct {
	CheckOnly      bool                          // 仅检查版本，不执行升级
	Force          bool                          // 强制升级，即使版本相同
	CurrentVersion string                        // 当前版本
	ProgressFunc   func(downloaded, total int64) // 下载进度回调
	StatusFunc     func(status string)           // 状态更新回调
	Context        context.Context
}

// UpgradeResult 升级结果
type UpgradeResult struct {
	NeedUpgrade    bool
	CurrentVersion string
	LatestVersion  string
	Success        bool
	Error          error
	Message        string
}

// CheckForUpdate 检查更新
func CheckForUpdate(currentVersion string) (*UpgradeResult, error) {
	// 加载 manifest
	manifest, err := LoadManifestFromURL(meta.ManifestURL)
	if err != nil {
		return nil, i18n.NewError("error.upgrade.check_failed", nil, err)
	}

	if manifest.LatestVersion == "" {
		return nil, i18n.NewError("error.manifest.latest_missing", nil, nil)
	}

	// 解析版本
	current, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, i18n.NewError("error.upgrade.current_version_invalid", map[string]any{"Version": currentVersion}, err)
	}

	latest, err := ParseVersion(manifest.LatestVersion)
	if err != nil {
		return nil, i18n.NewError("error.upgrade.latest_version_invalid", map[string]any{"Version": manifest.LatestVersion}, err)
	}

	// 比较版本
	needUpgrade := latest.IsNewerThan(current)

	return &UpgradeResult{
		NeedUpgrade:    needUpgrade,
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
		Success:        true,
	}, nil
}

// PerformUpgrade 执行升级
func PerformUpgrade(opts UpgradeOptions) *UpgradeResult {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	updateStatus := func(status string) {
		if opts.StatusFunc != nil {
			opts.StatusFunc(status)
		}
		logs.Infof(status)
	}

	// 1. 检查更新
	updateStatus(i18n.T("upgrade.status.checking"))
	result, err := CheckForUpdate(opts.CurrentVersion)
	if err != nil {
		return &UpgradeResult{
			Success: false,
			Error:   err,
			Message: i18n.T("upgrade.result.check_failed", map[string]any{"Cause": err}),
		}
	}

	result.CurrentVersion = opts.CurrentVersion

	// 仅检查模式
	if opts.CheckOnly {
		if result.NeedUpgrade {
			result.Message = i18n.T("upgrade.result.available", map[string]any{"Latest": result.LatestVersion, "Current": result.CurrentVersion})
		} else {
			result.Message = i18n.T("upgrade.result.current", map[string]any{"Version": result.CurrentVersion})
		}
		result.Success = true
		return result
	}

	// 检查是否需要升级
	if !result.NeedUpgrade && !opts.Force {
		result.Message = i18n.T("upgrade.result.current", map[string]any{"Version": result.CurrentVersion})
		result.Success = true
		return result
	}

	// 2. 加载 manifest
	updateStatus(i18n.T("upgrade.status.fetching_release"))
	manifest, err := LoadManifestFromURL(meta.ManifestURL)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.release_failed", map[string]any{"Cause": err})
		return result
	}

	// 3. 获取当前平台的二进制文件信息
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binary, err := manifest.GetBinaryForPlatform(goos, goarch)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.platform_failed", map[string]any{"Cause": err})
		return result
	}

	// 4. 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.executable_failed", map[string]any{"Cause": err})
		return result
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.executable_resolve_failed", map[string]any{"Cause": err})
		return result
	}

	// 5. 下载新版本到临时文件
	updateStatus(i18n.T("upgrade.status.downloading", map[string]any{"Version": result.LatestVersion}))
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, binary.Filename+".tmp")

	err = DownloadFile(DownloadFileOptions{
		URL:          binary.URL,
		DestPath:     tempFile,
		ProgressFunc: opts.ProgressFunc,
		Context:      ctx,
	})
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.download_failed", map[string]any{"Cause": err})
		return result
	}
	defer os.Remove(tempFile) // 清理临时文件

	// 6. 校验文件完整性
	updateStatus(i18n.T("upgrade.status.verifying"))
	if err := verifyChecksum(tempFile, binary.Checksum); err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.verify_failed", map[string]any{"Cause": err})
		return result
	}

	// 7. 备份当前版本
	updateStatus(i18n.T("upgrade.status.backing_up"))
	backupPath := execPath + meta.UpgradeBackupSuffix
	if err := copyFile(execPath, backupPath); err != nil {
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.backup_failed", map[string]any{"Cause": err})
		return result
	}

	// 8. 替换可执行文件
	updateStatus(i18n.T("upgrade.status.installing"))
	if err := replaceExecutable(tempFile, execPath); err != nil {
		// 替换失败，尝试回滚
		logs.Errorf("Installation failed; rolling back: %v", err)
		if rollbackErr := copyFile(backupPath, execPath); rollbackErr != nil {
			result.Success = false
			result.Error = i18n.NewError("error.upgrade.install_and_rollback_failed", map[string]any{"Rollback": rollbackErr}, err)
			result.Message = i18n.T("upgrade.result.manual_recovery")
			return result
		}
		result.Success = false
		result.Error = err
		result.Message = i18n.T("upgrade.result.install_rolled_back", map[string]any{"Cause": err})
		return result
	}

	// 9. 删除备份文件
	_ = os.Remove(backupPath)

	result.Success = true
	result.Message = i18n.T("upgrade.result.success", map[string]any{"Current": result.CurrentVersion, "Latest": result.LatestVersion})
	return result
}

// verifyChecksum 校验文件 SHA256
func verifyChecksum(filePath, expectedChecksum string) error {
	// 移除 "sha256:" 前缀（如果有）
	expectedChecksum = strings.TrimPrefix(expectedChecksum, "sha256:")

	file, err := os.Open(filePath)
	if err != nil {
		return i18n.NewError("error.io.open_file_failed", map[string]any{"Path": filePath}, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return i18n.NewError("error.io.hash_failed", map[string]any{"Path": filePath}, err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return i18n.NewError("error.upgrade.checksum_mismatch", map[string]any{"Expected": expectedChecksum, "Actual": actualChecksum}, nil)
	}

	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// 获取源文件信息（用于保留权限）
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// replaceExecutable 替换可执行文件
func replaceExecutable(newFile, targetPath string) error {
	// 读取目标文件的权限
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return i18n.NewError("error.io.stat_failed", map[string]any{"Path": targetPath}, err)
	}

	// 删除旧文件
	if err := os.Remove(targetPath); err != nil {
		return i18n.NewError("error.io.delete_failed", map[string]any{"Path": targetPath}, err)
	}

	// 复制新文件
	if err := copyFile(newFile, targetPath); err != nil {
		return i18n.NewError("error.io.copy_failed", map[string]any{"Source": newFile, "Destination": targetPath}, err)
	}

	// 恢复执行权限
	if err := os.Chmod(targetPath, targetInfo.Mode()); err != nil {
		return i18n.NewError("error.io.permissions_failed", map[string]any{"Path": targetPath}, err)
	}

	return nil
}

// CalculateFileSHA256 计算文件的 SHA256 哈希值
func CalculateFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
