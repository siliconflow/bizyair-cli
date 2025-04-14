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
	"path"
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

// Deprecated: manually upload a single file
func UploadFile(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdUploadFile)
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
		sha256sum, md5Hash, err := CalculateHash(fileToUpload.Path)
		if err != nil {
			return err
		}

		fileToUpload.Signature = sha256sum
		logs.Debugf(fmt.Sprintf("file: %s, signature: %s\n", fileToUpload.RelPath, fileToUpload.Signature))

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
			ossClient, err := lib.NewAliOssStorageClient(fileStorage.Region, fileStorage.Endpoint, fileStorage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
			if err != nil {
				return err
			}

			_, err = ossClient.UploadFile(fileToUpload, fileRecord.ObjectKey, fileIndex)
			if err != nil {
				return err
			}

			// commit
			_, err = client.CommitFileV2(fileToUpload.Signature, fileRecord.ObjectKey, md5Hash, args.Type)

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

// upload a structured model folder
func upload(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdUpload)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// init client
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

	// check model path
	folders, err := checkPath(args)
	if err != nil {
		return err
	}

	if len(folders) > 1 {
		return cli.Exit("uploading multiple folders is not supported yet", meta.LoadError)
	}
	modelPath := folders[0]
	stat, err := os.Stat(modelPath)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return cli.Exit("uploading non-directory is not supported yet", meta.LoadError)
	}

	// parse model name
	nameFile := path.Join(modelPath, meta.ModelNameFileName)
	readName, err := os.ReadFile(nameFile)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to get model name, error: %s", err), meta.LoadError)
	}
	modelName := string(readName)
	fmt.Printf("Model name: %s\n", modelName)

	modelVersionList := make([]lib.ModelVersion, 0)

	// parse model type
	typeFile := path.Join(modelPath, meta.ModelTypeFileName)
	readType, err := os.ReadFile(typeFile)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to get model type, error: %s", err), meta.LoadError)
	}
	modelType := string(readType)
	// check type
	checkType := meta.UploadFileType(modelType)
	if !lo.Contains[meta.UploadFileType](meta.ModelTypes, checkType) {
		return cli.Exit(fmt.Sprintf("Unsupported type [%s], only works for %s", args.Type, meta.ModelTypesStr), meta.LoadError)
	}

	// parse model versions:	path/to/folder/{version name}	info: description.md, content.json, basemodel.txt
	candidateVersionDir, err := filepath.Glob(modelPath + "/*")
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to scan model versions, %s", err), meta.LoadError)
	}
	for _, versionDir := range candidateVersionDir {
		relPath, _ := filepath.Rel(modelPath, versionDir)
		stat, err := os.Stat(versionDir)
		if err != nil {
			logs.Warnf("skip: failed to read file or dir: %s, error: %s\n", versionDir, err)
		}
		if stat.IsDir() {
			fmt.Printf("start to parse folder: %s\n", versionDir)
			modelVersion, err := parseModelVersion(*client, modelPath, versionDir)
			if err != nil {
				logs.Warnf("skip folder: [%s]\n", relPath)
			} else {
				logs.Debugf("added version folder: %s\n", versionDir)
				modelVersionList = append(modelVersionList, *modelVersion)
			}
		}
	}
	if len(modelVersionList) == 0 {
		return cli.Exit(fmt.Sprintf("no valid version scanned in path: %s\n", modelPath), meta.LoadError)
	}

	// if cover exists in root dir, assign it to the first version
	coverList, _ := filepath.Glob(fmt.Sprintf("%s/%s", modelPath, meta.CoverFileName))
	if len(coverList) != 0 {
		for idx, cover := range coverList {
			logs.Debugf("uploading cover : %s, index: %d/%d\n", &cover, idx, len(coverList))
			coverUrl, err := client.UploadImageToOss(cover)
			if err != nil {
				logs.Errorf("failed to upload cover to oss\n")
				continue
			}
			modelVersionList[0].CoverUrls = append(modelVersionList[0].CoverUrls, coverUrl)
		}
	}

	ValidVersions := make([]*lib.ModelVersion, 0)

	// commit version content
	for _, modelVersion := range modelVersionList {
		stat, err := os.Stat(modelVersion.Path)
		if err != nil {
			logs.Errorf("failed to upload version: [%s], error: %s\n", modelVersion.Version, err)
			continue
		}

		relPath, err := filepath.Rel(filepath.Dir(modelVersion.Path), modelVersion.Path)
		if err != nil {
			logs.Errorf("failed to upload version: [%s], error: %s\n", modelVersion.Version, err)
			continue
		}
		uploadFile := lib.FileToUpload{
			Path:    filepath.ToSlash(modelVersion.Path),
			RelPath: filepath.ToSlash(relPath),
			Size:    stat.Size(),
		}

		err = signAndCommit(client, &uploadFile, modelType)
		if err != nil {
			logs.Errorf("failed to upload version: [%s], error: %s\n", modelVersion.Version, err)
			continue // skip current version
		}
		modelVersion.Sign = uploadFile.Signature
		ValidVersions = append(ValidVersions, &modelVersion)
	}

	// commit model
	_, err = client.CommitModelV2(modelName, modelType, ValidVersions)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to commit model, error: %s\n", err), meta.LoadError)
	}
	logs.Debugf("successfully upload model: %s\n", modelName)
	return nil
}

// parse a folder, upload content and return version info.
func parseModelVersion(client lib.Client, modelPath string, versionPath string) (*lib.ModelVersion, error) {
	/**
	Version:		${dirName}
	Path:			./content.* (required)
	Introduction:	./description.md
	Sign:			(nil)
	BaseModel:		default: other
	Public:			default: true
	CoverUrls:
	*/

	// get version name
	versionName, err := filepath.Rel(modelPath, versionPath)
	if err != nil {
		logs.Warnf("failed to parse version name for path: %s\n", versionPath)
		return nil, err
	}

	// get version content
	contentFileList, err := filepath.Glob(fmt.Sprintf("%s/%s", versionPath, meta.ContentFileName))
	if err != nil {
		logs.Warnf("failed to fetch version content in current path: %s\n", versionPath)
		return nil, err
	}
	if len(contentFileList) == 0 {
		logs.Warnf("version content not found in current path: %s\n", versionPath)
		return nil, err
	}
	if len(contentFileList) > 1 {
		logs.Warnf("multiple content files found in current path: %s\n", versionPath)
		return nil, err
	}
	contentFile := contentFileList[0]
	logs.Debugf("version: [%s] scanned\n", versionName)

	// parse BaseModel
	baseModel := string(meta.TypeOther)
	baseModelList, err := filepath.Glob(fmt.Sprintf("%s/%s", versionPath, meta.BaseModelFileName))
	if err != nil {
		logs.Warnf("failed to match basemodel file, set `other` by default\n")
	}
	if len(baseModelList) != 0 {
		baseModelPath := baseModelList[0] // only use the first BaseModel file
		readBaseModel, err := os.ReadFile(baseModelPath)
		if err != nil {
			logs.Warnf("failed to read base model file: %s\n", baseModelPath)
		} else {
			baseModel = string(readBaseModel)
		}
		valid, exists := meta.SupportedBaseModels[baseModel]
		if !exists || !valid {
			logs.Errorf("unsupported base model: %s\n", baseModel)
			return nil, err
		}
	}

	// parse introduction
	introText := ""
	introFileList, err := filepath.Glob(fmt.Sprintf("%s/%s", versionPath, meta.IntroFileName))
	if err != nil {
		logs.Warnf("failed to match introduction file for version: [%s], set empty by default\n", versionName)
	}
	if len(introFileList) != 0 {
		introPath := introFileList[0] // only use the first description file
		readIntro, err := os.ReadFile(introPath)
		if err != nil {
			logs.Warnf("failed to read intro file: %s\n", introPath)
		} else {
			introText = string(readIntro)
		}
	}

	// parse covers
	coverList, err := filepath.Glob(fmt.Sprintf("%s/%s", versionPath, meta.CoverFileName))
	if err != nil {
		logs.Warnf("failed to fetch cover for version: [%s]\n", versionName)
	}
	coverUrlList := make([]string, 0)
	// tempWebpList := make([]string, 0)
	for idx, cover := range coverList {
		logs.Debugf("uploading cover : %s, index: %d/%d\n", &cover, idx, len(coverList))
		coverUrl, err := client.UploadImageToOss(cover)
		if err != nil {
			logs.Errorf("failed to upload cover to oss\n")
			continue
		}
		coverUrlList = append(coverUrlList, coverUrl)
	}

	// parse public flag
	public := false
	publicFiles, err := filepath.Glob(fmt.Sprintf("%s/%s", versionPath, meta.PublicFileName))
	if err != nil {
		logs.Warnf("failed to read public flag, set false by default\n")
	}
	if len(publicFiles) != 0 {
		publicFile := publicFiles[0]
		readPublic, err := os.ReadFile(publicFile)
		if err != nil {
			logs.Warnf("failed to read public flag, set false by default\n")
		}
		value, err := strconv.ParseBool(string(readPublic))
		if err != nil {
			logs.Warnf("failed to parse public flag: %s, set false by default\n", string(readPublic))
		} else {
			public = value
		}
	}

	versionInfo := lib.ModelVersion{
		Version:      versionName,
		Path:         contentFile, // remain abs path because here the file is not uploaded yet
		Introduction: introText,
		BaseModel:    baseModel,
		CoverUrls:    coverUrlList,
		Public:       public,
	}
	return &versionInfo, nil
}

func signAndCommit(client *lib.Client, fileToUpload *lib.FileToUpload, Type string) error {
	// 1. calculate file hash
	sha256sum, md5Hash, err := CalculateHash(fileToUpload.Path)
	if err != nil {
		return err
	}

	fileToUpload.Signature = sha256sum
	fileToUpload.Md5Hash = md5Hash
	logs.Debugf(fmt.Sprintf("file: %s, signature: %s\n", fileToUpload.RelPath, fileToUpload.Signature))

	// 2. pass sha256sum to the server
	ossCert, err := client.OssSign(sha256sum, Type)
	if err != nil {
		return err
	}

	fileIndex := fmt.Sprintf("%d/%d", 1, 1)

	fileRecord := ossCert.Data.File
	if fileRecord.Id == 0 {
		// upload file
		fileStorage := ossCert.Data.Storage
		ossClient, err := lib.NewAliOssStorageClient(fileStorage.Region, fileStorage.Endpoint, fileStorage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
		if err != nil {
			return err
		}

		_, err = ossClient.UploadFile(fileToUpload, fileRecord.ObjectKey, fileIndex)

		// commit
		_, err = client.CommitFileV2(fileToUpload.Signature, fileRecord.ObjectKey, fileToUpload.Md5Hash, Type)

		if err != nil {
			return err
		}
	} else {
		// skip
		fileToUpload.Id = fileRecord.Id
		fileToUpload.RemoteKey = fileRecord.ObjectKey

		fmt.Fprintln(os.Stdout, fmt.Sprintf("(%s) %s Already Uploaded", fileIndex, fileToUpload.RelPath))
	}
	return nil
}

// CalculateHash calculates the SHA256 hash of a file.
func CalculateHash(filePath string) (string, string, error) {
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
		logs.Debugf("Base model not exists:%s\n", basemodel)
	}
	if exists && !valid {
		logs.Debugf("Base model not valid:%s\n", basemodel)
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
