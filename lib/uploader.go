package lib

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib/filehash"
)

// UploadOptions 上传选项
type UploadOptions struct {
	File         *FileToUpload    // 要上传的文件
	Client       *Client          // API 客户端
	ModelType    string           // 模型类型
	Context      context.Context  // 上下文（用于取消）
	ProgressFunc ProgressCallback // 进度回调函数
	FileIndex    string           // 文件索引（如 "1/3"）
}

// UnifiedUpload 统一上传逻辑（支持断点续传和分片上传）
// 返回上传后的 objectKey 和错误
func UnifiedUpload(opts UploadOptions) (string, error) {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// 1. 获取文件信息
	st, err := os.Stat(opts.File.Path)
	if err != nil {
		return "", WithStep("读取文件信息", err)
	}
	if st.IsDir() {
		return "", WithStep("校验路径", fmt.Errorf("仅支持文件上传，不支持目录: %s", opts.File.Path))
	}

	// 确保文件大小已设置
	if opts.File.Size == 0 {
		opts.File.Size = st.Size()
	}

	// 2. 计算文件哈希
	sha256sum, md5Hash, err := filehash.CalculateHash(opts.File.Path)
	if err != nil {
		return "", WithStep("计算哈希", err)
	}
	opts.File.Signature = sha256sum

	logs.Debugf("[%s] 文件哈希: %s\n", opts.FileIndex, sha256sum)

	// 3. 尝试加载断点续传信息
	resumed := false
	checkpointFile, _ := GetCheckpointFile(sha256sum)
	checkpoint, _ := LoadCheckpoint(checkpointFile)

	var ossClient *AliOssStorageClient
	var objectKey string

	// 4. 如果有有效的 checkpoint，尝试续传
	if checkpoint != nil && ValidateCheckpoint(checkpoint, opts.File) {
		logs.Debugf("[%s] 发现有效的 checkpoint，准备续传\n", opts.FileIndex)

		// 检查 checkpoint 中的凭证是否可用且未过期
		if checkpoint.AccessKeyId != "" && checkpoint.AccessKeySecret != "" && !IsCredentialExpired(checkpoint.Expiration) {
			logs.Debugf("[%s] checkpoint 凭证有效，使用缓存凭证\n", opts.FileIndex)
			cli, oerr := NewAliOssStorageClient(
				checkpoint.Endpoint, checkpoint.Bucket,
				checkpoint.AccessKeyId, checkpoint.AccessKeySecret,
				checkpoint.SecurityToken,
			)
			if oerr == nil {
				cli.SetExpiration(checkpoint.Expiration)
				ossClient = cli
				objectKey = checkpoint.ObjectKey
			} else {
				logs.Warnf("[%s] 使用 checkpoint 凭证失败: %v，将刷新凭证\n", opts.FileIndex, oerr)
			}
		} else {
			// 凭证过期或不存在：删除 checkpoint，重新上传
			logs.Warnf("[%s] checkpoint 凭证已过期，删除 checkpoint 并重新上传\n", opts.FileIndex)
			if checkpointFile != "" {
				_ = DeleteCheckpoint(checkpointFile)
			}
			checkpoint = nil // 重置为 nil，后续流程会走全新上传
		}
	}

	// 5. 如果没有有效的 OSS 客户端，获取新的签名
	if ossClient == nil {
		ossCert, err := opts.Client.OssSign(sha256sum, opts.ModelType)
		if err != nil {
			return "", WithStep("获取上传签名", err)
		}

		fileRecord := ossCert.Data.File

		// 文件已存在于服务器，直接跳过
		if fileRecord.Id > 0 {
			logs.Debugf("[%s] 文件已存在，跳过上传\n", opts.FileIndex)
			opts.File.Id = fileRecord.Id
			opts.File.RemoteKey = fileRecord.ObjectKey

			// 如果有进度回调，显示 100%
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(opts.File.Size, opts.File.Size)
			}

			return fileRecord.ObjectKey, nil
		}

		// 创建新的 OSS 客户端
		storage := ossCert.Data.Storage
		cli, err := NewAliOssStorageClient(
			storage.Endpoint, storage.Bucket,
			fileRecord.AccessKeyId, fileRecord.AccessKeySecret,
			fileRecord.SecurityToken,
		)
		if err != nil {
			return "", WithStep("创建OSS客户端", err)
		}
		cli.SetExpiration(fileRecord.Expiration)
		ossClient = cli
		objectKey = fileRecord.ObjectKey
	}

	// 6. 使用分片上传（自动支持断点续传）
	_, err = ossClient.UploadFileMultipart(ctx, opts.File, objectKey, opts.FileIndex, opts.ProgressFunc)
	if err != nil {
		// 检查是否是用户取消
		if errors.Is(err, context.Canceled) {
			return "", err // 直接返回不包装，保持 context.Canceled 类型
		}
		return "", WithStep("OSS上传", err)
	}

	// 7. 提交文件
	commitKey := objectKey
	if opts.File.RemoteKey != "" {
		commitKey = opts.File.RemoteKey
	}
	_, err = opts.Client.CommitFileV2(sha256sum, commitKey, md5Hash, opts.ModelType)
	if err != nil {
		return "", WithStep("提交文件", err)
	}

	logs.Debugf("[%s] 上传成功: %s\n", opts.FileIndex, objectKey)
	resumed = true // 标记为成功

	if resumed {
		return commitKey, nil
	}
	return objectKey, nil
}
