package lib

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/schollz/progressbar/v3"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type AliOssStorageClient struct {
	ossClient        *oss.Client
	ossBucketName    string
	ossRegion        string
	ossBucket        *oss.Bucket
	ossSecurityToken string
}

type AliOssStorageProvider struct {
	Cred oss.Credentials
}

func (a *AliOssStorageProvider) GetCredentials() oss.Credentials {
	return a.Cred
}

type AliOssStorageCred struct {
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
}

func (c *AliOssStorageCred) GetAccessKeyID() string {
	return c.AccessKeyId
}

func (c *AliOssStorageCred) GetAccessKeySecret() string {
	return c.AccessKeySecret
}

func (c *AliOssStorageCred) GetSecurityToken() string {
	return c.SecurityToken
}

type FileToUpload struct {
	Id        int64
	Path      string
	RelPath   string
	Size      int64
	Signature string
	RemoteKey string
	Md5Hash	  string
}

type OssProgressListener struct {
	File      *FileToUpload
	Bar       *progressbar.ProgressBar
	Throttled func(x int)
	FileIndex string
}

func (listener *OssProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		bar := progressbar.DefaultBytes(
			event.TotalBytes,
			fmt.Sprintf("(%s) %s", listener.FileIndex, filepath.Base(listener.File.RelPath)),
		)
		listener.Bar = bar
		listener.Throttled = Throttle(func(x int) {
			listener.Bar.Set(x)
		}, time.Millisecond*500)
		listener.Bar.Set(int(event.ConsumedBytes))
	case oss.TransferDataEvent:
		if event.TotalBytes != 0 {
			listener.Throttled(int(event.ConsumedBytes))
		}
	case oss.TransferCompletedEvent:
		listener.Bar.Set(int(event.ConsumedBytes))
	case oss.TransferFailedEvent:
		listener.Bar.Set(int(event.ConsumedBytes))
	default:
	}
}

// FIXME: 这里ossRegion传入的值应该类似于`oss-cn-shanghai`，而实际传入了`oss-cn-shanghai.aliyun.com`，
// 导致后面解析oss url失败，这里endpoint需要用于构造ossClient，所以这里多加一个入参region
func NewAliOssStorageClient(region, endpoint, bucketName, accessKey, secretKey, securityToken string) (*AliOssStorageClient, error) {
	provider := AliOssStorageProvider{
		Cred: &AliOssStorageCred{
			AccessKeyId:     accessKey,
			AccessKeySecret: secretKey,
			SecurityToken:   securityToken,
		},
	}
	ossClient, err := oss.New(endpoint, "", "", oss.SetCredentialsProvider(&provider))
	if err != nil {
		return nil, cli.Exit(fmt.Errorf("failed to init oss client: %s", err), meta.LoadError)
	}

	bucket, err := ossClient.Bucket(bucketName)
	if err != nil {
		return nil, cli.Exit(fmt.Errorf("failed to get bucket: %s", err), meta.LoadError)
	}

	ossStorageClient := &AliOssStorageClient{
		ossClient:        ossClient,
		ossBucketName:    bucketName,
		ossBucket:        bucket,
		ossRegion:        region,
		ossSecurityToken: securityToken,
	}

	logs.Debugf("new oss storage client: %v", ossStorageClient)
	return ossStorageClient, nil
}

func (a *AliOssStorageClient) UploadFile(file *FileToUpload, objectName string, fileIndex string) (string, error) {
	err := a.ossBucket.PutObjectFromFile(objectName, file.Path, oss.ObjectACL(oss.ACLPublicRead), oss.Progress(&OssProgressListener{
		File:      file,
		FileIndex: fileIndex,
	}))

	if err != nil {
		logs.Errorf("Failed to upload file", "error", err)
		return "", err
	}
	return fmt.Sprintf(meta.OSSObjectKey, a.ossBucketName, a.ossRegion, objectName), nil
}

func (a *AliOssStorageClient) MultipartUpload(filePath string, objectName string) (string, error) {
	chunks, err := oss.SplitFileByPartSize(filePath, 1024*1024)
	fd, err := os.Open(filePath)
	defer fd.Close()

	imur, err := a.ossBucket.InitiateMultipartUpload(objectName)
	var options []oss.Option
	for _, chunk := range chunks {
		options = append(options, oss.AddParam("partNumber", strconv.Itoa(chunk.Number)))
		options = append(options, oss.AddParam("uploadId", imur.UploadID))
		// 生成签名URL。
		signedURL, err := a.ossBucket.SignURL(objectName, oss.HTTPPut, 600, options...)
		if err != nil {
			return "", cli.Exit(fmt.Errorf("failed to list uploaded parts: %s", err), meta.LoadError)
		}
		logs.Debugf("signature url: %s", signedURL)
	}

	lsRes, err := a.ossBucket.ListUploadedParts(imur)
	if err != nil {
		return "", cli.Exit(fmt.Errorf("failed to list uploaded parts: %s", err), meta.LoadError)
	}

	// 遍历分片，并填充ETag值。
	var parts []oss.UploadPart
	for _, p := range lsRes.UploadedParts {
		parts = append(parts, oss.UploadPart{XMLName: p.XMLName, PartNumber: p.PartNumber, ETag: p.ETag})
	}

	_, err = a.ossBucket.CompleteMultipartUpload(imur, parts)
	if err != nil {
		return "", cli.Exit(fmt.Errorf("failed to complete multipart upload: %s", err), meta.LoadError)
	}

	return fmt.Sprintf(meta.OSSObjectKey, a.ossBucketName, a.ossRegion, objectName), nil
}
