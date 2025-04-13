package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/cmd"
	"github.com/siliconflow/bizyair-cli/lib"
)

func main() {
	Run()
	// test()
}

func Run() {
	defer func() {
		logs.Flush()
	}()

	cli := cmd.Init()
	err := cli.Run(os.Args)
	if err != nil {
		logs.Errorf("%v\n", err)
	}
}

func test() {
	// https://bizyair-api.siliconflow.cn
	// https://uat-api.bizyair.cn
	Domain := "https://uat-api.bizyair.cn"
	ApiKey := "sk-nrbhtyyetwmlkyjcnnhpdvbggrgpbezpeasgguqgtjqnzqil"
	client := lib.NewClient(Domain, ApiKey)
	imgPath := "C:\\Users\\92489\\Downloads\\17622049\\cover.gif"
	//imgName := "cover.gif"

	_, webpImgName, _, err := lib.ImageToWebp(imgPath)
	if err != nil {
		logs.Errorf("%v\n", err)
		return
	}

	stat, err := os.Stat(imgPath)
	if err != nil {
		logs.Errorf("failed to parse image path: %s", imgPath)
		return
	}

	fileUrl := url.PathEscape(webpImgName)
	// serverUrl := fmt.Sprintf("%s/upload/token?file_name=%s", Domain, fileName)
	resp, err := client.Get_upload_token(fileUrl, "image")
	if err != nil {
		logs.Errorf("failed to get upload token, %s", err)
		return
	}
	fileRecord := resp.Data.File
	fileStorage := resp.Data.Storage
	ossClient, err := lib.NewAliOssStorageClient(fileStorage.Region, fileStorage.Endpoint, fileStorage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
	if err != nil {
		return
	}
	fileToUpload := lib.FileToUpload{
		Path:      webpImgName,
		RelPath:   webpImgName,
		Size:      stat.Size(),
		RemoteKey: fileRecord.ObjectKey,
	}
	result, err := ossClient.UploadFile(&fileToUpload, fileRecord.ObjectKey, "1/1")
	if err != nil {
		fmt.Printf("failed to upload image to oss, error: %s\n", err)
		return
	}
	fmt.Println("get response:", result)

}
