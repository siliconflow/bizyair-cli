package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/meta"
)

type AliOssStorageClient struct {
	ossClient        *oss.Client
	ossBucketName    string
	ossRegion        string
	ossSecurityToken string
	ossEndpoint      string
	ossAccessKeyId   string
	ossAccessKey     string
	ossExpiration    string
}

type FileToUpload struct {
	Id        int64
	Path      string
	RelPath   string
	Size      int64
	Signature string
	RemoteKey string
}

func parseRegionFromEndpoint(endpoint string) string {
	// 假设 endpoint 是像 "oss-cn-hangzhou.aliyuncs.com"
	// 解析出 "cn-hangzhou"
	// 如果不匹配，返回默认或错误
	if strings.Contains(endpoint, "oss-") && strings.Contains(endpoint, ".aliyuncs.com") {
		start := strings.Index(endpoint, "oss-") + 4
		end := strings.Index(endpoint[start:], ".")
		if end != -1 {
			return endpoint[start : start+end]
		}
	}
	// 如果解析失败，返回一个默认值或记录错误
	logs.Warnf("failed to parse region from endpoint: %s, using default", endpoint)
	return "cn-hangzhou" // 默认值
}

func NewAliOssStorageClient(endpoint, bucketName, accessKey, secretKey, securityToken string) (*AliOssStorageClient, error) {
	region := parseRegionFromEndpoint(endpoint)

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, securityToken)).
		WithRegion(region)

	client := oss.NewClient(cfg)

	ossStorageClient := &AliOssStorageClient{
		ossClient:        client,
		ossBucketName:    bucketName,
		ossRegion:        region,
		ossSecurityToken: securityToken,
		ossEndpoint:      endpoint,
		ossAccessKeyId:   accessKey,
		ossAccessKey:     secretKey,
		ossExpiration:    "",
	}

	logs.Debugf("new oss storage client: %v", ossStorageClient)
	return ossStorageClient, nil
}

// SetExpiration 设置当前临时凭证过期时间（用于写入 checkpoint）
func (a *AliOssStorageClient) SetExpiration(exp string) {
	a.ossExpiration = exp
}

func (a *AliOssStorageClient) UploadFile(file *FileToUpload, objectName string, fileIndex string, progress func(int64, int64)) (string, error) {
	return a.UploadFileCtx(context.TODO(), file, objectName, fileIndex, progress)
}

func (a *AliOssStorageClient) UploadFileCtx(ctx context.Context, file *FileToUpload, objectName string, fileIndex string, progress func(int64, int64)) (string, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %v", err)
	}

	totalSize := fileInfo.Size()
	logs.Debugf("file size: %d\n", totalSize)

	// 打开文件
	f, err := os.Open(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to open local file %v", err)
	}
	defer f.Close()

	// 创建进度追踪 reader
	var reader io.Reader = f
	if progress != nil {
		reader = &progressReader{
			reader:      f,
			total:       totalSize,
			progress:    progress,
			lastTime:    time.Now(),
			minInterval: 50 * time.Millisecond, // 更频繁的更新
		}
	}

	// 使用 PutObject 直接上传，支持流式进度
	putRequest := &oss.PutObjectRequest{
		Bucket: oss.Ptr(a.ossBucketName),
		Key:    oss.Ptr(objectName),
		Body:   reader,
	}

	_, err = a.ossClient.PutObject(ctx, putRequest)
	if err != nil {
		return "", fmt.Errorf("failed to put object %v", err)
	}

	// 确保进度回调显示100%
	if progress != nil {
		progress(totalSize, totalSize)
	}

	logs.Debugf("put object completed for %s\n", objectName)
	return fmt.Sprintf(meta.OSSObjectKey, a.ossBucketName, a.ossRegion, objectName), nil
}

// progressReader 包装 io.Reader 以提供进度回调
type progressReader struct {
	reader      io.Reader
	total       int64
	progress    func(int64, int64)
	readBytes   int64
	lastTime    time.Time
	minInterval time.Duration
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.readBytes += int64(n)
		now := time.Now()
		if now.Sub(pr.lastTime) >= pr.minInterval {
			pr.progress(pr.readBytes, pr.total)
			pr.lastTime = now
		}
	}
	return n, err
}

// UploadFileMultipart 使用分片上传方式上传文件（支持断点续传）
func (a *AliOssStorageClient) UploadFileMultipart(ctx context.Context, file *FileToUpload, objectName string, fileIndex string, progress func(int64, int64)) (string, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %v", err)
	}

	totalSize := fileInfo.Size()
	// 若调用方未填充 file.Size，这里补齐，避免校验失败
	if file.Size == 0 {
		file.Size = totalSize
	}
	logs.Debugf("[%s] file size: %d bytes (%.2f MB)\n", fileIndex, totalSize, float64(totalSize)/(1024*1024))

	// 判断是否使用分片上传
	if totalSize < meta.MultipartThreshold {
		logs.Debugf("[%s] file size < %d MB, using simple upload\n", fileIndex, meta.MultipartThreshold/(1024*1024))
		return a.UploadFileCtx(ctx, file, objectName, fileIndex, progress)
	}

	logs.Debugf("[%s] using multipart upload (size: %.2f MB)\n", fileIndex, float64(totalSize)/(1024*1024))

	// 生成checkpoint文件路径（仅按 sha256 命名）
	checkpointFile, err := GetCheckpointFile(file.Signature)
	if err != nil {
		logs.Warnf("[%s] failed to get checkpoint file: %v, will proceed without checkpoint\n", fileIndex, err)
		checkpointFile = ""
	}

	var uploadID string
	var checkpoint *CheckpointInfo
	var existingParts []oss.UploadPart

	// 尝试加载checkpoint（仅使用当前正确命名规则）
	if checkpointFile != "" {
		checkpoint, err = LoadCheckpoint(checkpointFile)
		if err != nil {
			logs.Warnf("[%s] failed to load checkpoint: %v\n", fileIndex, err)
			checkpoint = nil
		}

		// 验证checkpoint（不比对 objectKey）
		if checkpoint != nil && ValidateCheckpoint(checkpoint, file) {
			logs.Debugf("[%s] resuming upload from checkpoint (uploadID: %s)\n", fileIndex, checkpoint.UploadID)
			uploadID = checkpoint.UploadID
			// 强制采用 checkpoint 中记录的 ObjectKey，保持与前端逻辑一致
			if checkpoint.ObjectKey != "" {
				objectName = checkpoint.ObjectKey
				file.RemoteKey = objectName
			}
			// 如果 checkpoint 携带凭证，则优先用其重建 client，无需请求后端
			if checkpoint.AccessKeyId != "" && checkpoint.AccessKeySecret != "" {
				if cli, cerr := NewAliOssStorageClient(checkpoint.Endpoint, checkpoint.Bucket, checkpoint.AccessKeyId, checkpoint.AccessKeySecret, checkpoint.SecurityToken); cerr == nil {
					a.ossClient = cli.ossClient
					a.ossBucketName = checkpoint.Bucket
					a.ossRegion = parseRegionFromEndpoint(checkpoint.Endpoint)
					a.ossSecurityToken = checkpoint.SecurityToken
					a.ossEndpoint = checkpoint.Endpoint
					a.ossAccessKeyId = checkpoint.AccessKeyId
					a.ossAccessKey = checkpoint.AccessKeySecret
				} else {
					logs.Warnf("[%s] failed to rebuild client from checkpoint: %v\n", fileIndex, cerr)
				}
			}
			existingParts = checkpoint.UploadedParts
		} else if checkpoint != nil {
			logs.Warnf("[%s] checkpoint validation failed, starting new upload\n", fileIndex)
			// 旧文件若不匹配，删除之；不影响后续创建新上传
			if checkpointFile != "" {
				_ = DeleteCheckpoint(checkpointFile)
			}
			checkpoint = nil
		}
	}

	// 如果没有有效的checkpoint，初始化新的分片上传
	if uploadID == "" {
		initResult, err := a.initiateMultipartUpload(ctx, objectName)
		if err != nil {
			return "", fmt.Errorf("failed to initiate multipart upload: %v", err)
		}
		uploadID = *initResult.UploadId
		logs.Debugf("[%s] initiated new upload (uploadID: %s)\n", fileIndex, uploadID)

		// 创建新的checkpoint
		checkpoint = &CheckpointInfo{
			ObjectKey:       objectName,
			UploadID:        uploadID,
			FilePath:        file.Path,
			FileSize:        totalSize,
			FileSignature:   file.Signature,
			PartSize:        meta.MultipartPartSize,
			TotalParts:      (totalSize + meta.MultipartPartSize - 1) / meta.MultipartPartSize,
			UploadedParts:   []oss.UploadPart{},
			CreatedAt:       time.Now(),
			Bucket:          a.ossBucketName,
			Region:          a.ossRegion,
			Endpoint:        a.ossEndpoint,
			AccessKeyId:     a.ossAccessKeyId,
			AccessKeySecret: a.ossAccessKey,
			SecurityToken:   a.ossSecurityToken,
			Expiration:      a.ossExpiration,
		}
		// 保存当前使用的远端key，供上层在提交阶段复用
		file.RemoteKey = objectName
		// 立即保存一次 checkpoint（若启用）
		if checkpointFile != "" {
			_ = SaveCheckpoint(checkpoint)
		}
	}

	// 上传分片
	parts, err := a.uploadParts(ctx, file, objectName, uploadID, checkpoint, existingParts, fileIndex, totalSize, progress, checkpointFile)
	if err != nil {
		// 检查是否是用户取消
		if errors.Is(err, context.Canceled) {
			logs.Debugf("[%s] upload canceled by user, checkpoint saved for resuming\n", fileIndex)
			return "", err // 直接返回不包装，保持 context.Canceled 类型
		}

		// 检查是否是 NoSuchUpload 错误（UploadID已失效）
		errStr := err.Error()
		if strings.Contains(errStr, "NoSuchUpload") || strings.Contains(errStr, "does not exist") {
			logs.Warnf("[%s] uploadID is invalid or expired, deleting checkpoint and restarting...\n", fileIndex)
			// 删除无效的checkpoint
			if checkpointFile != "" {
				_ = DeleteCheckpoint(checkpointFile)
			}
			// 重新开始上传（递归调用一次）
			return a.UploadFileMultipart(ctx, file, objectName, fileIndex, progress)
		}

		// 其他错误：保留 checkpoint，便于下次自动断点续传；不调用 Abort
		logs.Warnf("[%s] upload failed, keep checkpoint for resuming: %v\n", fileIndex, err)
		return "", fmt.Errorf("failed to upload parts: %v", err)
	}

	// 完成分片上传
	_, err = a.completeMultipartUpload(ctx, objectName, uploadID, parts)
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %v", err)
	}

	// 确保进度回调显示100%
	if progress != nil {
		progress(totalSize, totalSize)
	}

	// 删除checkpoint文件
	if checkpointFile != "" {
		_ = DeleteCheckpoint(checkpointFile)
	}

	logs.Debugf("[%s] multipart upload completed: %s\n", fileIndex, objectName)
	return fmt.Sprintf(meta.OSSObjectKey, a.ossBucketName, a.ossRegion, objectName), nil
}

// initiateMultipartUpload 初始化分片上传
func (a *AliOssStorageClient) initiateMultipartUpload(ctx context.Context, objectKey string) (*oss.InitiateMultipartUploadResult, error) {
	request := &oss.InitiateMultipartUploadRequest{
		Bucket: oss.Ptr(a.ossBucketName),
		Key:    oss.Ptr(objectKey),
	}

	return a.ossClient.InitiateMultipartUpload(ctx, request)
}

// uploadParts 并发上传所有分片
func (a *AliOssStorageClient) uploadParts(
	ctx context.Context,
	file *FileToUpload,
	objectKey string,
	uploadID string,
	checkpoint *CheckpointInfo,
	existingParts []oss.UploadPart,
	fileIndex string,
	totalSize int64,
	progress func(int64, int64),
	checkpointFile string,
) ([]oss.UploadPart, error) {
	partSize := int64(meta.MultipartPartSize)
	partCount := (totalSize + partSize - 1) / partSize

	logs.Debugf("[%s] total parts: %d (part size: %.2f MB)\n", fileIndex, partCount, float64(partSize)/(1024*1024))

	// 打开文件
	f, err := os.Open(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	// 创建已上传分片的映射
	uploadedMap := make(map[int32]oss.UploadPart)
	for _, part := range existingParts {
		uploadedMap[part.PartNumber] = part
	}

	// 用于存储分片结果
	parts := make([]oss.UploadPart, partCount)
	var uploadedSize int64

	// 计算已上传的大小
	for _, part := range existingParts {
		idx := int(part.PartNumber - 1)
		if idx >= 0 && idx < int(partCount) {
			parts[idx] = part
			offset := int64(idx) * partSize
			currentPartSize := partSize
			if offset+currentPartSize > totalSize {
				currentPartSize = totalSize - offset
			}
			uploadedSize += currentPartSize
		}
	}

	// 更新初始进度
	if progress != nil {
		progress(uploadedSize, totalSize)
		if uploadedSize > 0 {
			logs.Debugf("[%s] resuming from %.1f%% (%d parts already uploaded)\n",
				fileIndex, float64(uploadedSize)*100/float64(totalSize), len(existingParts))
		}
	}

	// 使用信号量控制并发数
	sem := make(chan struct{}, meta.MultipartParallel)
	errChan := make(chan error, partCount)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并发上传分片
	for i := int64(0); i < partCount; i++ {
		partNumber := i + 1
		offset := i * partSize
		currentPartSize := partSize

		// 最后一个分片可能小于partSize
		if offset+currentPartSize > totalSize {
			currentPartSize = totalSize - offset
		}

		// 检查是否已上传
		if _, exists := uploadedMap[int32(partNumber)]; exists {
			logs.Debugf("[%s] part %d/%d already uploaded, skipping\n", fileIndex, partNumber, partCount)
			continue
		}

		wg.Add(1)

		go func(partNum int64, off int64, size int64) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			// 上传单个分片（带重试）
			part, err := a.uploadPartWithRetry(ctx, f, objectKey, uploadID, partNum, off, size, fileIndex, int(partCount))
			if err != nil {
				errChan <- fmt.Errorf("part %d failed: %v", partNum, err)
				return
			}

			// 保存分片信息
			mu.Lock()
			parts[partNum-1] = part
			uploadedSize += size

			// 更新checkpoint
			if checkpoint != nil && checkpointFile != "" {
				checkpoint.UploadedParts = make([]oss.UploadPart, 0)
				for _, p := range parts {
					if p.PartNumber > 0 {
						checkpoint.UploadedParts = append(checkpoint.UploadedParts, p)
					}
				}
				_ = SaveCheckpoint(checkpoint)
			}
			mu.Unlock()

			// 更新进度
			if progress != nil {
				progress(uploadedSize, totalSize)
			}

			logs.Debugf("[%s] part %d/%d uploaded (%.1f%%)\n",
				fileIndex, partNum, partCount, float64(uploadedSize)*100/float64(totalSize))

		}(partNumber, offset, currentPartSize)
	}

	// 等待所有分片上传完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if len(errChan) > 0 {
		err := <-errChan
		return nil, err
	}

	return parts, nil
}

// uploadPartWithRetry 上传单个分片（带重试逻辑）
func (a *AliOssStorageClient) uploadPartWithRetry(
	ctx context.Context,
	file *os.File,
	objectKey string,
	uploadID string,
	partNumber int64,
	offset int64,
	size int64,
	fileIndex string,
	totalParts int,
) (oss.UploadPart, error) {
	maxRetries := 3
	var lastErr error

	for retry := 0; retry <= maxRetries; retry++ {
		if retry > 0 {
			logs.Warnf("[%s] retrying part %d/%d (attempt %d/%d)\n", fileIndex, partNumber, totalParts, retry+1, maxRetries+1)
			time.Sleep(time.Second * time.Duration(retry)) // 指数退避
		}

		// 读取分片数据
		buffer := make([]byte, size)
		_, err := file.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			lastErr = err
			continue
		}

		// 上传分片
		request := &oss.UploadPartRequest{
			Bucket:     oss.Ptr(a.ossBucketName),
			Key:        oss.Ptr(objectKey),
			UploadId:   oss.Ptr(uploadID),
			PartNumber: int32(partNumber),
			Body:       bytes.NewReader(buffer),
		}

		result, err := a.ossClient.UploadPart(ctx, request)
		if err != nil {
			lastErr = err
			continue
		}

		// 上传成功
		return oss.UploadPart{
			PartNumber: int32(partNumber),
			ETag:       result.ETag,
		}, nil
	}

	return oss.UploadPart{}, fmt.Errorf("failed after %d retries: %v", maxRetries+1, lastErr)
}

// completeMultipartUpload 完成分片上传
func (a *AliOssStorageClient) completeMultipartUpload(
	ctx context.Context,
	objectKey string,
	uploadID string,
	parts []oss.UploadPart,
) (*oss.CompleteMultipartUploadResult, error) {
	request := &oss.CompleteMultipartUploadRequest{
		Bucket:   oss.Ptr(a.ossBucketName),
		Key:      oss.Ptr(objectKey),
		UploadId: oss.Ptr(uploadID),
		CompleteMultipartUpload: &oss.CompleteMultipartUpload{
			Parts: parts,
		},
	}

	return a.ossClient.CompleteMultipartUpload(ctx, request)
}

// abortMultipartUpload 中止分片上传
func (a *AliOssStorageClient) abortMultipartUpload(ctx context.Context, objectKey string, uploadID string) error {
	request := &oss.AbortMultipartUploadRequest{
		Bucket:   oss.Ptr(a.ossBucketName),
		Key:      oss.Ptr(objectKey),
		UploadId: oss.Ptr(uploadID),
	}

	_, err := a.ossClient.AbortMultipartUpload(ctx, request)
	return err
}
