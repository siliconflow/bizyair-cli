package lib

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
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
		return nil, fmt.Errorf("无法检查更新: %v", err)
	}

	if manifest.LatestVersion == "" {
		return nil, fmt.Errorf("manifest 中未找到最新版本")
	}

	// 解析版本
	current, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("无效的当前版本号: %v", err)
	}

	latest, err := ParseVersion(manifest.LatestVersion)
	if err != nil {
		return nil, fmt.Errorf("无效的最新版本号: %v", err)
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
	updateStatus("正在检查更新...")
	result, err := CheckForUpdate(opts.CurrentVersion)
	if err != nil {
		return &UpgradeResult{
			Success: false,
			Error:   err,
			Message: fmt.Sprintf("检查更新失败: %v", err),
		}
	}

	result.CurrentVersion = opts.CurrentVersion

	// 仅检查模式
	if opts.CheckOnly {
		if result.NeedUpgrade {
			result.Message = fmt.Sprintf("发现新版本: %s (当前版本: %s)", result.LatestVersion, result.CurrentVersion)
		} else {
			result.Message = fmt.Sprintf("已是最新版本: %s", result.CurrentVersion)
		}
		result.Success = true
		return result
	}

	// 检查是否需要升级
	if !result.NeedUpgrade && !opts.Force {
		result.Message = fmt.Sprintf("已是最新版本: %s", result.CurrentVersion)
		result.Success = true
		return result
	}

	// 2. 加载 manifest
	updateStatus("正在获取版本信息...")
	manifest, err := LoadManifestFromURL(meta.ManifestURL)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("获取版本信息失败: %v", err)
		return result
	}

	// 3. 获取当前平台的二进制文件信息
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binary, err := manifest.GetBinaryForPlatform(goos, goarch)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("获取平台信息失败: %v", err)
		return result
	}

	// 4. 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("获取可执行文件路径失败: %v", err)
		return result
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("解析可执行文件路径失败: %v", err)
		return result
	}

	// 5. 下载新版本到临时文件
	updateStatus(fmt.Sprintf("正在下载新版本 %s...", result.LatestVersion))
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
		result.Message = fmt.Sprintf("下载失败: %v", err)
		return result
	}
	defer os.Remove(tempFile) // 清理临时文件

	// 6. 校验文件完整性
	updateStatus("正在校验文件完整性...")
	if err := verifyChecksum(tempFile, binary.Checksum); err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("文件校验失败: %v", err)
		return result
	}

	// 7. 备份当前版本
	updateStatus("正在备份当前版本...")
	backupPath := execPath + meta.UpgradeBackupSuffix
	if err := copyFile(execPath, backupPath); err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("备份失败: %v", err)
		return result
	}

	// 8. 替换可执行文件
	updateStatus("正在安装新版本...")
	if err := replaceExecutable(tempFile, execPath); err != nil {
		// 替换失败，尝试回滚
		logs.Errorf("安装失败，正在回滚: %v", err)
		if rollbackErr := copyFile(backupPath, execPath); rollbackErr != nil {
			result.Success = false
			result.Error = fmt.Errorf("安装失败且回滚失败: %v, 回滚错误: %v", err, rollbackErr)
			result.Message = "升级失败，请手动恢复"
			return result
		}
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("安装失败，已回滚: %v", err)
		return result
	}

	// 9. 删除备份文件
	_ = os.Remove(backupPath)

	result.Success = true
	result.Message = fmt.Sprintf("✅ 升级成功！版本: %s -> %s", result.CurrentVersion, result.LatestVersion)
	return result
}

// verifyChecksum 校验文件 SHA256
func verifyChecksum(filePath, expectedChecksum string) error {
	// 移除 "sha256:" 前缀（如果有）
	expectedChecksum = strings.TrimPrefix(expectedChecksum, "sha256:")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("计算哈希失败: %v", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("校验和不匹配 (期望: %s, 实际: %s)", expectedChecksum, actualChecksum)
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
		return fmt.Errorf("无法读取目标文件信息: %v", err)
	}

	// 删除旧文件
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("无法删除旧文件: %v", err)
	}

	// 复制新文件
	if err := copyFile(newFile, targetPath); err != nil {
		return fmt.Errorf("无法复制新文件: %v", err)
	}

	// 恢复执行权限
	if err := os.Chmod(targetPath, targetInfo.Mode()); err != nil {
		return fmt.Errorf("无法设置执行权限: %v", err)
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
