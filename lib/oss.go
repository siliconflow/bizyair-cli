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
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

type AliOssStorageClient struct {
	ossClient     ossObjectClient
	ossBucketName string
	ossRegion     string
	ossEndpoint   string
}

type ossObjectClient interface {
	PutObject(context.Context, *oss.PutObjectRequest, ...func(*oss.Options)) (*oss.PutObjectResult, error)
	InitiateMultipartUpload(context.Context, *oss.InitiateMultipartUploadRequest, ...func(*oss.Options)) (*oss.InitiateMultipartUploadResult, error)
	UploadPart(context.Context, *oss.UploadPartRequest, ...func(*oss.Options)) (*oss.UploadPartResult, error)
	CompleteMultipartUpload(context.Context, *oss.CompleteMultipartUploadRequest, ...func(*oss.Options)) (*oss.CompleteMultipartUploadResult, error)
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
		ossClient:     client,
		ossBucketName: bucketName,
		ossRegion:     region,
		ossEndpoint:   endpoint,
	}

	logs.Debugf("new oss storage client: endpoint=%s bucket=%s region=%s", endpoint, bucketName, region)
	return ossStorageClient, nil
}

func (a *AliOssStorageClient) UploadFile(file *FileToUpload, objectName string, fileIndex string, progress func(int64, int64)) (string, error) {
	return a.UploadFileCtx(context.TODO(), file, objectName, fileIndex, progress)
}

func (a *AliOssStorageClient) UploadFileCtx(ctx context.Context, file *FileToUpload, objectName string, fileIndex string, progress func(int64, int64)) (string, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		return "", i18n.NewError("error.io.stat_failed", map[string]any{"Path": file.Path}, err)
	}

	totalSize := fileInfo.Size()
	logs.Debugf("file size: %d\n", totalSize)

	// 打开文件
	f, err := os.Open(file.Path)
	if err != nil {
		return "", i18n.NewError("error.io.open_file_failed", map[string]any{"Path": file.Path}, err)
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
		return "", i18n.NewError("error.oss.put_failed", map[string]any{"Object": objectName}, err)
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
	if ctx == nil {
		ctx = context.Background()
	}
	// 获取文件信息
	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		return "", i18n.NewError("error.io.stat_failed", map[string]any{"Path": file.Path}, err)
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
	return a.uploadFileMultipartWithRetry(ctx, file, objectName, fileIndex, totalSize, progress, checkpointFile)
}

const maxNoSuchUploadRestarts = 1

func (a *AliOssStorageClient) uploadFileMultipartWithRetry(
	ctx context.Context,
	file *FileToUpload,
	objectName string,
	fileIndex string,
	totalSize int64,
	progress func(int64, int64),
	checkpointFile string,
) (string, error) {
	for restart := 0; ; restart++ {
		result, err := a.uploadFileMultipartOnce(ctx, file, objectName, fileIndex, totalSize, progress, checkpointFile)
		if err == nil {
			return result, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !isNoSuchUpload(err) {
			return "", err
		}

		// The upload ID is unusable. Never leave it in a checkpoint, even when
		// the single automatic restart has already been consumed.
		if checkpointFile != "" {
			_ = DeleteCheckpoint(checkpointFile)
		}
		file.RemoteKey = ""
		if restart >= maxNoSuchUploadRestarts {
			return "", err
		}
		logs.Warnf("[%s] uploadID is invalid or expired; restarting multipart upload once\n", fileIndex)
	}
}

func (a *AliOssStorageClient) uploadFileMultipartOnce(
	ctx context.Context,
	file *FileToUpload,
	objectName string,
	fileIndex string,
	totalSize int64,
	progress func(int64, int64),
	checkpointFile string,
) (string, error) {

	var uploadID string
	var checkpoint *CheckpointInfo
	var existingParts []oss.UploadPart
	var err error

	// 尝试加载checkpoint（仅使用当前正确命名规则）
	if checkpointFile != "" {
		checkpoint, err = LoadCheckpoint(checkpointFile)
		if err != nil {
			logs.Warnf("[%s] failed to load checkpoint: %v\n", fileIndex, err)
			checkpoint = nil
		}

		// 验证checkpoint
		if checkpoint != nil && ValidateCheckpoint(checkpoint, file) {
			logs.Debugf("[%s] resuming upload from checkpoint (uploadID: %s)\n", fileIndex, checkpoint.UploadID)
			uploadID = checkpoint.UploadID
			// 强制采用 checkpoint 中记录的 ObjectKey，保持与前端逻辑一致
			if checkpoint.ObjectKey != "" {
				objectName = checkpoint.ObjectKey
				file.RemoteKey = objectName
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
			return "", i18n.NewError("error.oss.multipart_start_failed", map[string]any{"Object": objectName}, err)
		}
		uploadID = *initResult.UploadId
		logs.Debugf("[%s] initiated new upload (uploadID: %s)\n", fileIndex, uploadID)

		// 创建新的checkpoint
		checkpoint = &CheckpointInfo{
			ObjectKey:     objectName,
			UploadID:      uploadID,
			FilePath:      file.Path,
			FileSize:      totalSize,
			FileSignature: file.Signature,
			PartSize:      meta.MultipartPartSize,
			TotalParts:    (totalSize + meta.MultipartPartSize - 1) / meta.MultipartPartSize,
			UploadedParts: []oss.UploadPart{},
			CreatedAt:     time.Now(),
			Bucket:        a.ossBucketName,
			Region:        a.ossRegion,
			Endpoint:      a.ossEndpoint,
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

		// 其他错误：保留 checkpoint，便于下次自动断点续传；不调用 Abort
		logs.Warnf("[%s] upload failed, keep checkpoint for resuming: %v\n", fileIndex, err)
		return "", i18n.NewError("error.oss.parts_failed", map[string]any{"Object": objectName}, err)
	}

	// 完成分片上传
	_, err = a.completeMultipartUpload(ctx, objectName, uploadID, parts)
	if err != nil {
		return "", i18n.NewError("error.oss.multipart_complete_failed", map[string]any{"Object": objectName}, err)
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

func isNoSuchUpload(err error) bool {
	if err == nil {
		return false
	}
	var serviceErr *oss.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Code == "NoSuchUpload"
	}
	// Keep compatibility with wrapped responses from older OSS SDK versions.
	return strings.Contains(err.Error(), "NoSuchUpload")
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
	if ctx == nil {
		ctx = context.Background()
	}
	partSize := int64(meta.MultipartPartSize)
	partCount := (totalSize + partSize - 1) / partSize

	logs.Debugf("[%s] total parts: %d (part size: %.2f MB)\n", fileIndex, partCount, float64(partSize)/(1024*1024))

	// 打开文件
	f, err := os.Open(file.Path)
	if err != nil {
		return nil, i18n.NewError("error.io.open_file_failed", map[string]any{"Path": file.Path}, err)
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

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 使用信号量控制并发数
	sem := make(chan struct{}, meta.MultipartParallel)
	errChan := make(chan error, partCount)
	var wg sync.WaitGroup
	var stateMu sync.Mutex
	var progressMu sync.Mutex
	var checkpointSaveMu sync.Mutex
	lastReported := uploadedSize
	lastCheckpointSave := time.Now()
	var checkpointSequence uint64
	var latestCheckpointAttempt uint64

	checkpointSnapshotLocked := func(force bool) (*CheckpointInfo, uint64) {
		if checkpoint == nil || checkpointFile == "" {
			return nil, 0
		}
		now := time.Now()
		if !force && now.Sub(lastCheckpointSave) < time.Second {
			return nil, 0
		}
		lastCheckpointSave = now
		checkpointSequence++
		return cloneCheckpoint(checkpoint), checkpointSequence
	}
	persistCheckpoint := func(snapshot *CheckpointInfo, sequence uint64) {
		if snapshot == nil {
			return
		}
		checkpointSaveMu.Lock()
		defer checkpointSaveMu.Unlock()
		if sequence <= latestCheckpointAttempt {
			return
		}
		latestCheckpointAttempt = sequence
		if err := SaveCheckpoint(snapshot); err != nil {
			logs.Warnf("[%s] failed to save checkpoint: %v\n", fileIndex, err)
		}
	}
	reportProgress := func(consumed int64) {
		if progress == nil {
			return
		}
		progressMu.Lock()
		defer progressMu.Unlock()
		if consumed <= lastReported {
			return
		}
		progress(consumed, totalSize)
		lastReported = consumed
	}
	recordError := func(err error) {
		errChan <- err
		cancel()
	}

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

			// 获取信号量，同时允许首个失败立即取消等待中的分片。
			select {
			case sem <- struct{}{}:
			case <-workerCtx.Done():
				return
			}
			defer func() { <-sem }()

			// 上传单个分片（带重试）
			part, err := a.uploadPartWithRetry(workerCtx, f, objectKey, uploadID, partNum, off, size, fileIndex, int(partCount))
			if err != nil {
				// A sibling may already have failed and canceled workerCtx. In that
				// case the original error is already recorded.
				if workerCtx.Err() != nil && errors.Is(err, context.Canceled) {
					return
				}
				recordError(i18n.NewError("error.oss.part_failed", map[string]any{"Part": partNum}, err))
				return
			}

			// 保存分片信息
			stateMu.Lock()
			parts[partNum-1] = part
			uploadedSize += size
			snapshotSize := uploadedSize

			// 内存中 O(1) 追加，落盘节流并在锁外串行执行。
			if checkpoint != nil && checkpointFile != "" {
				checkpoint.UploadedParts = append(checkpoint.UploadedParts, part)
			}
			checkpointSnapshot, checkpointSeq := checkpointSnapshotLocked(false)
			stateMu.Unlock()
			persistCheckpoint(checkpointSnapshot, checkpointSeq)

			// 更新进度
			reportProgress(snapshotSize)

			logs.Debugf("[%s] part %d/%d uploaded (%.1f%%)\n",
				fileIndex, partNum, partCount, float64(snapshotSize)*100/float64(totalSize))

		}(partNumber, offset, currentPartSize)
	}

	// 等待所有分片上传完成
	wg.Wait()
	stateMu.Lock()
	checkpointSnapshot, checkpointSeq := checkpointSnapshotLocked(true)
	stateMu.Unlock()
	persistCheckpoint(checkpointSnapshot, checkpointSeq)
	close(errChan)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var uploadErrors []error
	for uploadErr := range errChan {
		uploadErrors = append(uploadErrors, uploadErr)
	}
	if len(uploadErrors) > 0 {
		return nil, errors.Join(uploadErrors...)
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
		if err := ctx.Err(); err != nil {
			return oss.UploadPart{}, err
		}
		if retry > 0 {
			logs.Warnf("[%s] retrying part %d/%d (attempt %d/%d)\n", fileIndex, partNumber, totalParts, retry+1, maxRetries+1)
			timer := time.NewTimer(time.Second * time.Duration(retry))
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return oss.UploadPart{}, ctx.Err()
			case <-timer.C:
			}
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
			if isNoSuchUpload(err) {
				return oss.UploadPart{}, err
			}
			continue
		}

		// 上传成功
		return oss.UploadPart{
			PartNumber: int32(partNumber),
			ETag:       result.ETag,
		}, nil
	}

	return oss.UploadPart{}, i18n.NewError("error.oss.part_retries_exhausted", map[string]any{"Attempts": maxRetries + 1, "Part": partNumber}, lastErr)
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
