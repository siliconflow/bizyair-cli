package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/cmd/hz/util"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
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

	// 1. 校验参数
	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	if err := lib.ValidateModelName(args.Name); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	modelPath, err := checkPath(args)
	if err != nil {
		return err
	}

	// 2. 准备文件列表
	var filesToUpload []*lib.FileToUpload
	for _, mPath := range modelPath {
		stat, err := os.Stat(mPath)
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		if stat.IsDir() {
			return cli.Exit("uploading directory is not supported yet", meta.LoadError)
		}

		relPath, err := filepath.Rel(filepath.Dir(mPath), mPath)
		if err != nil {
			relPath = filepath.Base(mPath)
		}
		filesToUpload = append(filesToUpload, &lib.FileToUpload{
			Path:    filepath.ToSlash(mPath),
			RelPath: filepath.ToSlash(relPath),
			Size:    stat.Size(),
		})
	}

	total := len(filesToUpload)
	if total < 1 {
		return cli.Exit("No files found to upload", meta.LoadError)
	}

	// 3. 处理版本信息
	modelVersion, err := getVersion(args)
	if err != nil {
		return err
	}
	versionPublic, err := checkPublic(args, false)
	if err != nil {
		return err
	}
	modelIntro, err := checkModelIntro(args, false)
	if err != nil {
		return err
	}

	// 4. 校验基础模型
	for _, base := range args.BaseModel {
		if err := lib.ValidateBaseModel(base); err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 4.5 校验封面（必填）
	if len(args.CoverUrls) == 0 {
		return cli.Exit("封面是必填项，请使用 -cover 标志为每个版本指定封面", meta.LoadError)
	}
	if len(args.CoverUrls) != len(args.Path) {
		return cli.Exit(fmt.Errorf("封面数量（%d）必须与文件数量（%d）相同", len(args.CoverUrls), len(args.Path)), meta.LoadError)
	}
	for idx, cover := range args.CoverUrls {
		if strings.TrimSpace(cover) == "" {
			return cli.Exit(fmt.Errorf("版本 %d 的封面不能为空", idx+1), meta.LoadError)
		}
	}

	// 5. 获取 API Key
	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	client := lib.NewClient(args.BaseDomain, apiKey)

	// 6. 预检查模型名是否已存在
	exists, err := client.CheckModelExists(args.Name, args.Type)
	if err != nil {
		return cli.Exit(fmt.Errorf("检查模型名失败: %w", err), meta.LoadError)
	}
	if exists && !args.Overwrite {
		return cli.Exit(fmt.Errorf("模型名 '%s' 已存在，请使用不同的名称或添加 --overwrite 标志", args.Name), meta.LoadError)
	}

	// 7. 开始并发上传
	fmt.Fprintf(os.Stdout, "开始上传 %d 个文件（并发数：3）\n", total)

	ctx := context.Background()
	versionList := make([]*lib.ModelVersion, total)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // 最多 3 个并发
	var mu sync.Mutex
	var uploadErrors []error

	for i, fileToUpload := range filesToUpload {
		wg.Add(1)
		idx := i
		file := fileToUpload

		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileIndex := fmt.Sprintf("%d/%d", idx+1, total)

			// 使用统一上传函数
			_, err := lib.UnifiedUpload(lib.UploadOptions{
				File:      file,
				Client:    client,
				ModelType: args.Type,
				Context:   ctx,
				FileIndex: fileIndex,
				ProgressFunc: func(consumed, total int64) {
					if total > 0 {
						percent := float64(consumed) / float64(total)
						bar := renderProgressBar(percent)
						fmt.Printf("\r(%s) %s %s %.1f%% (%s/%s)",
							fileIndex,
							filepath.Base(file.RelPath),
							bar,
							percent*100,
							format.FormatBytes(consumed),
							format.FormatBytes(total))

						if percent >= 1.0 {
							fmt.Println() // 上传完成后换行
						}
					}
				},
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Errorf("[%s] %w", fileIndex, err))
				mu.Unlock()
				return
			}

			// 处理封面上传（必填）
			var coverUrls []string
			coverUrl, cerr := lib.UploadCover(client, args.CoverUrls[idx], ctx)
			if cerr != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Errorf("[%s] 封面上传失败: %w", fileIndex, cerr))
				mu.Unlock()
				return
			}
			if coverUrl != "" {
				coverUrls = []string{coverUrl}
			}

			// 构建版本信息
			mu.Lock()
			versionList[idx] = &lib.ModelVersion{
				Version:      modelVersion[idx],
				BaseModel:    args.BaseModel[idx],
				Introduction: modelIntro[idx],
				Public:       versionPublic[idx],
				Sign:         file.Signature,
				Path:         modelPath[idx],
				CoverUrls:    coverUrls,
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 8. 检查错误
	if len(uploadErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n上传失败，错误列表：\n")
		for _, err := range uploadErrors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return cli.Exit("部分文件上传失败", meta.ServerError)
	}

	// 9. 提交模型
	_, err = client.CommitModelV2(args.Name, args.Type, versionList)
	if err != nil {
		return cli.Exit(fmt.Errorf("提交模型失败: %w", err), meta.ServerError)
	}

	fmt.Fprintf(os.Stdout, "\n✓ 上传成功！\n")
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
