package lib

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
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
	}

	logs.Debugf("new oss storage client: %v", ossStorageClient)
	return ossStorageClient, nil
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
