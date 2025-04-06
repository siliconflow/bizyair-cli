package cmd

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc64"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/hertz/cmd/hz/util"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/samber/lo"
	"github.com/siliconflow/siliconcloud-cli/config"
	"github.com/siliconflow/siliconcloud-cli/lib"
	"github.com/siliconflow/siliconcloud-cli/meta"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdUpload)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	if err = checkType(args, true); err != nil {
		return err
	}

	if err = checkName(args, true); err != nil {
		return err
	}

	modelPath, err := checkPath(args)
	if err != nil {
		return err
	}

	stat, err := os.Stat(modelPath)
	if err != nil {
		return cli.Exit(fmt.Errorf("check path failed: %s", err), meta.LoadError)
	}

	// version: default 'v1'
	modelVersion := getVersion(args)

	_, err = checkModelIntro(args, false)
	if err != nil {
		return err
	}

	cover_urls, err := checkCoverUrl(args, false)
	if err != nil {
		return err
	}

	if err := checkBaseModel(args, false); err != nil {
		return err
	}

	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return err
		}
	}

	client := lib.NewClient(args.BaseDomain, apiKey)

	// modelExistResp, err := client.CheckModel(args.Type, args.Name)
	// if err != nil {
	// 	return err
	// }

	// if modelExistResp.Data.Exists {
	// 	if !args.Overwrite {
	// 		return cli.Exit(fmt.Sprintf("Model already exists, use --overwrite to overwrite it"), meta.LoadError)
	// 	}
	// }

	var filesToUpload []*lib.FileToUpload

	// TODO: upload dir is not supported
	if stat.IsDir() {
		return cli.Exit("uploading directory is not supported yet", meta.LoadError)
		// recursively upload all files in the directory
		// err = filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
		// 	if err != nil {
		// 		return err
		// 	}

		// 	for _, uploadPath := range meta.IgnoreUploadDirs {
		// 		if filepath.Base(path) == uploadPath {
		// 			if info.IsDir() {
		// 				return filepath.SkipDir
		// 			}
		// 		}
		// 	}

		// 	if !info.IsDir() {
		// 		// calculate file relative path
		// 		relPath, err := filepath.Rel(modelPath, path)
		// 		if err != nil {
		// 			return err
		// 		}

		// 		filesToUpload = append(filesToUpload, &lib.FileToUpload{
		// 			Path:    filepath.ToSlash(path),
		// 			RelPath: filepath.ToSlash(relPath),
		// 			Size:    info.Size(),
		// 		})
		// 	}

		// 	return err
		// })

		// if err != nil {
		// 	return cli.Exit(fmt.Errorf("traverse dir failed: %s", err), meta.LoadError)
		// }
	} else {
		// 上传单个文件
		relPath, err := filepath.Rel(filepath.Dir(modelPath), modelPath)
		if err != nil {
			return err
		}
		filesToUpload = append(filesToUpload, &lib.FileToUpload{
			Path:    filepath.ToSlash(modelPath),
			RelPath: filepath.ToSlash(relPath),
			Size:    stat.Size(),
		})
	}

	total := len(filesToUpload)
	if total < 1 {
		return cli.Exit("No files found to upload, you cannot upload an empty directory!", meta.LoadError)
	}
	if total > 1 {
		return cli.Exit("Uploadig multiple files is not supported yet!", meta.LoadError)
	}

	// start to upload files
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Start uploading %d files", total))

	// TODO: upload multiple files is not supported
	for i, fileToUpload := range filesToUpload {
		// calculate file hash
		sha256sum, md5_hash, err := calculateHash(fileToUpload.Path)
		if err != nil {
			return err
		}

		fileToUpload.Signature = sha256sum
		logs.Debugf(fmt.Sprintf("file: %s, signature: %s", fileToUpload.RelPath, fileToUpload.Signature))

		// pass sha256sum to the server
		ossCert, err := client.OssSign(sha256sum, args.Type)
		if err != nil {
			return err
		}

		fileIndex := fmt.Sprintf("%d/%d", i+1, total)

		fileRecord := ossCert.Data.File
		if fileRecord.Id == 0 {
			// upload file
			fileStorage := ossCert.Data.Storage
			ossClient, err := lib.NewAliOssStorageClient(fileStorage.Endpoint, fileStorage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
			if err != nil {
				return err
			}

			_, err = ossClient.UploadFile(fileToUpload, fileRecord.ObjectKey, fileIndex)
			if err != nil {
				return err
			}

			// commit
			_, err = client.CommitFileV2(fileToUpload.Signature, fileRecord.ObjectKey, md5_hash, args.Type)

			if err != nil {
				return err
			}
			// fileToUpload.Id = newFileRecord.Data.File.Id
			// fileToUpload.RemoteKey = newFileRecord.Data.File.ObjectKey
		} else {
			// skip
			fileToUpload.Id = fileRecord.Id
			fileToUpload.RemoteKey = fileRecord.ObjectKey

			fmt.Fprintln(os.Stdout, fmt.Sprintf("(%s) %s Already Uploaded", fileIndex, fileToUpload.RelPath))
		}
		versionInfo := lib.ModelVersion{
			Version:      modelVersion,
			BaseModel:    args.BaseModel,
			Introduction: args.Intro,
			Public:       args.Public,
			Sign:         sha256sum,
			Path:         modelPath,
			CoverUrls:    cover_urls,
		}
		versionList := []*lib.ModelVersion{&versionInfo}

		_, err = client.CommitModelV2(args.Name, args.Type, versionList)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "Uploaded successfully\n")

	return nil
}

// calculateHash calculates the SHA256 hash of a file.
func calculateHash(filePath string) (string, string, error) {
	// read file and calculate CRC64
	tabECMA := crc64.MakeTable(crc64.ECMA)
	hashCRC := crc64.New(tabECMA)
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	io.Copy(hashCRC, file)
	crc1 := hashCRC.Sum64()

	// reset file pointer to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", "", err
	}

	// MD5
	hashMD5 := md5.New()
	io.Copy(hashMD5, file)
	md5Str := base64.StdEncoding.EncodeToString(hashMD5.Sum(nil))

	// 计算SHA256
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s%d", md5Str, crc1)))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	logs.Debugf("file: %s, crc64: %d, md5: %s, sha256: %s", filePath, crc1, md5Str, hashString)
	return hashString, md5Str, nil
}

func checkType(args *config.Argument, required bool) error {
	if required && args.Type == "" {
		return cli.Exit("The following arguments are required: type", meta.LoadError)
	}

	if args.Type == "" {
		return nil
	}

	modelType := meta.UploadFileType(args.Type)
	if !lo.Contains[meta.UploadFileType](meta.ModelTypes, modelType) {
		return cli.Exit(fmt.Sprintf("Unsupported type, only works for %s", meta.ModelTypesStr), meta.LoadError)
	}

	return nil
}

func checkPath(args *config.Argument) (string, error) {
	if args.Path == "" {
		return "", cli.Exit("The following arguments are required: path", meta.LoadError)
	}

	exists, err := util.PathExist(args.Path)
	if err != nil {
		return "", cli.Exit(fmt.Errorf("check path failed: %s", err), meta.LoadError)
	}
	if !exists {
		return "", cli.Exit(fmt.Sprintf("path %s does not exist", args.Path), meta.LoadError)
	}

	if !filepath.IsAbs(args.Path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", cli.Exit(fmt.Errorf("get current path failed: %s", err), meta.LoadError)
		}
		ap := filepath.Join(cwd, args.Path)
		return ap, nil
	}

	return args.Path, nil
}

func checkName(args *config.Argument, required bool) error {
	if required && args.Name == "" {
		return cli.Exit("The following arguments are required: name", meta.LoadError)
	}

	if args.Name == "" {
		return nil
	}

	re := regexp.MustCompile(`^[\w-]+$`)
	matches := re.MatchString(args.Name)
	if !matches {
		return cli.Exit(fmt.Errorf("invalid \"name\", it can only include numbers, English letters, and \"-\" or \"_\""), meta.LoadError)
	}
	return nil
}

func getVersion(args *config.Argument) string {
	defaultVersion := "v1"
	if args.ModelVersion != "" {
		return args.ModelVersion
	}
	return defaultVersion
}

func checkBaseModel(args *config.Argument, required bool) error {
	if required && args.BaseModel == "" {
		return cli.Exit("The following arguments are required: base_model", meta.LoadError)
	}
	isValid := isValidBaseModel(args.BaseModel)
	if !isValid {
		return cli.Exit("Base model not supported: "+args.BaseModel, meta.LoadError)
	}
	return nil
}

func checkCoverUrl(args *config.Argument, required bool) ([]string, error) {
	if required && args.CoverUrls == "" {
		return nil, cli.Exit("The following arguments are required: cover_urls", meta.LoadError)
	}
	if args.CoverUrls == "" {
		return nil, nil
	}
	urlArray := strings.Split(args.CoverUrls, ",")

	return urlArray, nil
}

func checkModelIntro(args *config.Argument, required bool) (string, error) {
	if required && args.Intro == "" {
		return "", cli.Exit("The following arguments are required: intro", meta.LoadError)
	}
	return args.Intro, nil
}

func isValidBaseModel(basemodel string) bool {
	valid, exists := meta.SupportedBaseModels[basemodel]
	if !exists {
		logs.Debugf("Base model not exists: ", basemodel)
	}
	if exists && !valid {
		logs.Debugf("Base model not valid: ", basemodel)
	}
	return valid
}
