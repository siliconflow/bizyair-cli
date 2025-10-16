package lib

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// UploadCover 统一封面上传逻辑（支持 URL 和本地文件）
// 返回上传后的 OSS URL
func UploadCover(client *Client, coverInput string, ctx context.Context) (string, error) {
	if coverInput == "" {
		return "", nil
	}

	// 去除分号后的多余 URL（只保留第一个）
	if idx := strings.Index(coverInput, ";"); idx >= 0 {
		coverInput = strings.TrimSpace(coverInput[:idx])
	}

	coverInput = strings.TrimSpace(coverInput)
	if coverInput == "" {
		return "", nil
	}

	localPath := coverInput
	var cleanup func()

	// 1. 如果是 HTTP URL，下载到临时文件
	if IsHTTPURL(coverInput) {
		p, cfn, err := DownloadToTemp(coverInput)
		if err != nil {
			return "", WithStep("封面下载", fmt.Errorf("下载失败: %s, %v", coverInput, err))
		}
		localPath = p
		cleanup = cfn
		defer cleanup()
	}

	// 2. 校验封面文件格式和大小（视频限 100MB）
	if err := ValidateCoverFile(localPath); err != nil {
		return "", WithStep("封面校验", err)
	}

	// 3. 获取上传凭证
	token, err := client.GetUploadToken(filepath.Base(localPath), "inputs")
	if err != nil {
		return "", WithStep("封面凭证", fmt.Errorf("获取上传凭证失败: %s, %v", coverInput, err))
	}

	fileRec := token.Data.File
	storage := token.Data.Storage

	// 4. 创建 OSS 客户端并上传
	ossCli, err := NewAliOssStorageClient(storage.Endpoint, storage.Bucket,
		fileRec.AccessKeyId, fileRec.AccessKeySecret, fileRec.SecurityToken)
	if err != nil {
		return "", WithStep("封面OSS客户端", fmt.Errorf("创建 OSS 客户端失败: %s, %v", coverInput, err))
	}

	coverFile := &FileToUpload{
		Path:    localPath,
		RelPath: filepath.Base(localPath),
		Size:    0, // 会在上传时自动获取
	}

	_, err = ossCli.UploadFileCtx(ctx, coverFile, fileRec.ObjectKey, "", nil)
	if err != nil {
		return "", WithStep("封面上传", fmt.Errorf("上传 OSS 失败: %s, %v", coverInput, err))
	}

	// 5. 提交并获取可用 URL
	commit, err := client.CommitInputResource(filepath.Base(localPath), fileRec.ObjectKey)
	if err != nil {
		return "", WithStep("封面提交", fmt.Errorf("提交失败: %s, %v", coverInput, err))
	}

	if commit == nil || commit.Data.Url == "" {
		return "", WithStep("封面提交", fmt.Errorf("未获取到有效 URL"))
	}

	return commit.Data.Url, nil
}
