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
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/cmd/hz/util"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/samber/lo"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
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

	var filesToUpload []*lib.FileToUpload

	for _, mPath := range modelPath {
		stat, err := os.Stat(mPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return cli.Exit("uploading directory is not supported yet", meta.LoadError)
		}

		relPath, err := filepath.Rel(filepath.Dir(mPath), mPath)
		if err != nil {
			return err
		}
		filesToUpload = append(filesToUpload, &lib.FileToUpload{
			Path:    filepath.ToSlash(mPath),
			RelPath: filepath.ToSlash(relPath),
			Size:    stat.Size(),
		})
	}

	total := len(filesToUpload)
	if total < 1 {
		return cli.Exit("No files found to upload, you cannot upload an empty directory!", meta.LoadError)
	}

	// version: default 'v{idx}'
	modelVersion, err := getVersion(args)
	if err != nil {
		return err
	}
	versionPublic, err := checkPublic(args, false)
	if err != nil {
		return nil
	}

	modelIntro, err := checkModelIntro(args, false)
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


	// 	// TODO: overwrite model

	// start to upload files
	fmt.Fprintln(os.Stdout, fmt.Sprintf("Start uploading %d files", total))

	versionList := make([]*lib.ModelVersion, 0)
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
		} else {
			// skip
			fileToUpload.Id = fileRecord.Id
			fileToUpload.RemoteKey = fileRecord.ObjectKey

			fmt.Fprintln(os.Stdout, fmt.Sprintf("(%s) %s Already Uploaded", fileIndex, fileToUpload.RelPath))
		}
		versionInfo := lib.ModelVersion{
			Version:      modelVersion[i],
			BaseModel:    args.BaseModel[i],
			Introduction: modelIntro[i],
			Public:       versionPublic[i],
			Sign:         sha256sum,
			Path:         modelPath[i],
			CoverUrls:    cover_urls[i],
		}
		versionList = append(versionList, &versionInfo)

	}
	// upload versions
	_, err = client.CommitModelV2(args.Name, args.Type, versionList)
	if err != nil {
		return err
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

	modelType := meta.UploadFileType(args.Type)
	if !lo.Contains[meta.UploadFileType](meta.ModelTypes, modelType) {
		return cli.Exit(fmt.Sprintf("Unsupported type [%s], only works for %s", args.Type, meta.ModelTypesStr), meta.LoadError)
	}

	return nil
}

func checkPath(args *config.Argument) ([]string, error) {
	if len(args.Path) == 0 {
		return nil, cli.Exit("The following arguments are required: path", meta.LoadError)
	}
	for idx, aPath := range args.Path {
		exists, err := util.PathExist(aPath)
		if err != nil {
			return nil, cli.Exit(fmt.Errorf("check path failed: %s", err), meta.LoadError)
		}
		if !exists {
			return nil, cli.Exit(fmt.Sprintf("path %s does not exist", args.Path), meta.LoadError)
		}
		if !filepath.IsAbs(aPath) {
			cwd, err := os.Getwd()
			if err != nil {
				return nil, cli.Exit(fmt.Errorf("get current path failed: %s", err), meta.LoadError)
			}
			args.Path[idx] = filepath.Join(cwd, aPath)
		}
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

func getVersion(args *config.Argument) ([]string, error) {
	defaultVersion := make([]string, 0)
	if len(args.ModelVersion) == 0 {
		for idx := range len(args.Path) {
			defaultVersion = append(defaultVersion, fmt.Sprintf("v%d", idx))
		}
		return defaultVersion, nil
	}
	if len(args.ModelVersion) == len(args.Path) {
		return args.ModelVersion, nil
	}

	return nil, cli.Exit(fmt.Sprintf("Required %d version arguments, but got %d", len(args.Path), len(args.ModelVersion)), meta.LoadError)
}

func checkPublic(args *config.Argument, required bool) ([]bool, error) {
	publicList, err := parseBoolStringSlice(args.VersionPublic)
	if err != nil {
		return nil, cli.Exit(fmt.Sprintf("Parse version bool error: %s", err.Error()), meta.LoadError)
	}
	if len(args.VersionPublic) == 0 && required {
		return nil, cli.Exit("The following arguments are required: vpub", meta.LoadError)
	}
	if len(args.VersionPublic) == 0 {
		return make([]bool, len(args.Path)), nil
	}
	if len(args.VersionPublic) != len(args.Path) {
		return nil, cli.Exit(fmt.Sprintf("Required %d vpub arguments, but got %d", len(args.Path), len(args.VersionPublic)), meta.LoadError)
	}
	return publicList, nil
}

func checkBaseModel(args *config.Argument, required bool) error {
	if required && len(args.BaseModel) == 0 {
		return cli.Exit("The following arguments are required: base_model", meta.LoadError)
	}
	if len(args.BaseModel) != len(args.Path) {
		cli.Exit(fmt.Sprintf("Required %d BaseModel arguments, but got %d", len(args.Path), len(args.ModelVersion)), meta.LoadError)
	}
	for _, base := range args.BaseModel {
		isValid := isValidBaseModel(base)
		if !isValid {
			return cli.Exit(fmt.Sprintf("Base model not supported: [%s]", args.BaseModel), meta.LoadError)
		}
	}

	return nil
}

func checkCoverUrl(args *config.Argument, required bool) ([][]string, error) {
	urlList := make([][]string, 0)
	if required && len(args.CoverUrls) != len(args.Path) {
		return nil, cli.Exit(fmt.Sprintf("Required %d cover_url arguments, but got %d", len(args.Path), len(args.CoverUrls)), meta.LoadError)
	}
	for idx := range len(args.Path) {
		if idx >= len(args.CoverUrls) {
			urlList = append(urlList, nil)
		} else {
			urlList = append(urlList, strings.Split(args.CoverUrls[idx], ";"))
		}
	}
	return urlList, nil
}

func checkModelIntro(args *config.Argument, required bool) ([]string, error) {
	Intro := make([]string, 0)
	if required && len(args.Intro) != len(args.Path) {
		return nil, cli.Exit(fmt.Sprintf("Required %d intro arguments, but got %d", len(args.Path), len(args.Intro)), meta.LoadError)
	}
	for idx := range len(args.Path) {
		if idx >= len(args.Intro) {
			Intro = append(Intro, "")
		} else {
			Intro = append(Intro, args.Intro[idx])
		}
	}
	return Intro, nil
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

func parseBoolStringSlice(s []string) ([]bool, error) {
	result := make([]bool, len(s))
	for idx, str := range s {
		b, err := strconv.ParseBool(str)
		if err != nil {
			return nil, fmt.Errorf("invalid bool string %s at index = %d", str, idx)
		}
		result[idx] = b
	}
	return result, nil
}
