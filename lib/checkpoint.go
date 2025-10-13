package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/meta"
)

// CheckpointInfo 断点续传信息
type CheckpointInfo struct {
	ObjectKey     string           `json:"object_key"`     // OSS对象键
	UploadID      string           `json:"upload_id"`      // 分片上传ID
	FilePath      string           `json:"file_path"`      // 本地文件路径
	FileSize      int64            `json:"file_size"`      // 文件大小
	FileSignature string           `json:"file_signature"` // 文件SHA256签名
	PartSize      int64            `json:"part_size"`      // 分片大小
	TotalParts    int64            `json:"total_parts"`    // 总分片数
	UploadedParts []oss.UploadPart `json:"uploaded_parts"` // 已上传的分片列表
	CreatedAt     time.Time        `json:"created_at"`     // 创建时间

	// 用于恢复续传的凭证与存储信息
	Bucket          string `json:"bucket,omitempty"`
	Region          string `json:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyId     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	SecurityToken   string `json:"security_token,omitempty"`
	Expiration      string `json:"expiration,omitempty"` // RFC3339 时间
}

// GetCheckpointDir 获取checkpoint目录路径 (~/.siliconflow/uploads/)
func GetCheckpointDir() (string, error) {
	var homeDir string
	currentOS := runtime.GOOS

	if currentOS == meta.OSWindows {
		homeDir = os.Getenv(meta.EnvUserProfile)
	} else {
		homeDir = os.Getenv(meta.EnvHome)
	}

	if homeDir == "" {
		return "", fmt.Errorf("unable to determine home directory")
	}

	checkpointDir := filepath.Join(homeDir, meta.SfFolder, meta.CheckpointFolder)

	// 确保目录存在
	if err := os.MkdirAll(checkpointDir, 0770); err != nil {
		return "", fmt.Errorf("failed to create checkpoint directory: %v", err)
	}

	return checkpointDir, nil
}

// GetCheckpointFile 根据文件签名生成checkpoint文件路径（仅按sha256命名）
func GetCheckpointFile(sha256sum string) (string, error) {
	checkpointDir, err := GetCheckpointDir()
	if err != nil {
		return "", err
	}

	// 使用 .checkpoint 扩展名
	checkpointFile := filepath.Join(checkpointDir, sha256sum+".checkpoint")
	return checkpointFile, nil
}

// SaveCheckpoint 保存断点信息到JSON文件
func SaveCheckpoint(info *CheckpointInfo) error {
	if info.FileSignature == "" {
		return fmt.Errorf("file signature is empty, cannot save checkpoint")
	}

	checkpointFile, err := GetCheckpointFile(info.FileSignature)
	if err != nil {
		return fmt.Errorf("failed to get checkpoint file path: %v", err)
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint info: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(checkpointFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %v", err)
	}

	logs.Debugf("checkpoint saved: %s\n", checkpointFile)
	return nil
}

// LoadCheckpoint 从JSON文件加载断点信息
func LoadCheckpoint(checkpointFile string) (*CheckpointInfo, error) {
	// 检查文件是否存在
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		return nil, nil // 文件不存在，返回nil但不报错
	}

	// 读取文件
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %v", err)
	}

	// 反序列化
	var info CheckpointInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint info: %v", err)
	}

	logs.Debugf("checkpoint loaded: %s\n", checkpointFile)
	return &info, nil
}

// DeleteCheckpoint 删除checkpoint文件
func DeleteCheckpoint(checkpointFile string) error {
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		return nil // 文件不存在，不需要删除
	}

	if err := os.Remove(checkpointFile); err != nil {
		return fmt.Errorf("failed to delete checkpoint file: %v", err)
	}

	logs.Debugf("checkpoint deleted: %s\n", checkpointFile)
	return nil
}

// ValidateCheckpoint 验证checkpoint是否有效（不比对 objectKey，仅校验文件与分片大小）
func ValidateCheckpoint(info *CheckpointInfo, file *FileToUpload) bool {
	if info == nil {
		return false
	}

	// 检查文件路径是否一致
	if info.FilePath != file.Path {
		logs.Warnf("checkpoint validation failed: file path mismatch (expected: %s, got: %s)\n", info.FilePath, file.Path)
		return false
	}

	// 检查文件大小是否一致
	if info.FileSize != file.Size {
		logs.Warnf("checkpoint validation failed: file size mismatch (expected: %d, got: %d)\n", info.FileSize, file.Size)
		return false
	}

	// 检查文件签名是否一致
	if info.FileSignature != file.Signature {
		logs.Warnf("checkpoint validation failed: file signature mismatch\n")
		return false
	}

	// 检查必要字段
	if info.UploadID == "" || info.ObjectKey == "" {
		logs.Warnf("checkpoint validation failed: missing upload ID or object key\n")
		return false
	}

	// 校验分片大小一致
	if info.PartSize != int64(meta.MultipartPartSize) {
		logs.Warnf("checkpoint validation failed: part size changed (cp: %d, now: %d)\n", info.PartSize, meta.MultipartPartSize)
		return false
	}

	logs.Debugf("checkpoint validation passed\n")
	return true
}
