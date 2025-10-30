package lib

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// UploadCover 统一封面上传逻辑（支持 URL 和本地文件）
// 返回上传后的 OSS URL
// statusCallback: 可选的状态回调函数，用于通知封面处理状态
func UploadCover(client *Client, coverInput string, ctx context.Context, statusCallback func(status, message string)) (string, error) {
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

	// 2.5. 如果是图片，转换为WebP格式
	if statusCallback != nil {
		statusCallback("converting", "封面转换中...")
	}

	webpPath, webpCleanup, err := ConvertImageToWebP(localPath)
	convertFailed := false
	if err != nil {
		// 转换失败，回退使用原格式（不中断上传）
		webpPath = localPath
		convertFailed = true
		if statusCallback != nil {
			statusCallback("fallback", "转换失败，使用原格式")
		}
	} else {
		if webpCleanup != nil {
			defer webpCleanup()
		}
		if statusCallback != nil && webpPath != localPath {
			statusCallback("ready", "封面已准备")
		}
	}

	// 使用转换后的文件路径（或原路径）
	uploadPath := localPath
	uploadFileName := filepath.Base(localPath)
	if webpPath != localPath && !convertFailed {
		uploadPath = webpPath
		// 更新文件名为 .webp 扩展名
		baseName := strings.TrimSuffix(filepath.Base(localPath), filepath.Ext(localPath))
		uploadFileName = baseName + ".webp"
	}

	// 3. 获取上传凭证
	token, err := client.GetUploadToken(uploadFileName, "inputs")
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
		Path:    uploadPath,
		RelPath: uploadFileName,
		Size:    0, // 会在上传时自动获取
	}

	_, err = ossCli.UploadFileCtx(ctx, coverFile, fileRec.ObjectKey, "", nil)
	if err != nil {
		return "", WithStep("封面上传", fmt.Errorf("上传 OSS 失败: %s, %v", coverInput, err))
	}

	// 5. 提交并获取可用 URL
	commit, err := client.CommitInputResource(uploadFileName, fileRec.ObjectKey)
	if err != nil {
		return "", WithStep("封面提交", fmt.Errorf("提交失败: %s, %v", coverInput, err))
	}

	if commit == nil || commit.Data.Url == "" {
		return "", WithStep("封面提交", fmt.Errorf("未获取到有效 URL"))
	}

	if statusCallback != nil {
		statusCallback("done", "封面已上传")
	}

	return commit.Data.Url, nil
}
