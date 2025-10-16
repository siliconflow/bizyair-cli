package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// checkModelExists 检查模型名是否已存在
func checkModelExists(apiKey, modelName, modelType string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		exists, err := client.CheckModelExists(modelName, modelType)
		return checkModelExistsDoneMsg{exists: exists, err: err}
	}
}

// loadBaseModelTypes 从后端加载基础模型类型列表
func loadBaseModelTypes(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		resp, err := client.GetBaseModelTypes()
		if err != nil {
			return baseModelTypesLoadedMsg{items: nil, err: err}
		}
		return baseModelTypesLoadedMsg{items: resp.Data, err: nil}
	}
}

// 多版本上传
func runUploadActionMulti(u uploadInputs, versions []versionItem) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan tea.Msg, 64)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer close(ch)

			args := config.NewArgument()
			args.Type = u.typ
			args.Name = u.name

			// 验证所有版本都有封面（必填）
			for i, v := range versions {
				if strings.TrimSpace(v.cover) == "" {
					ch <- actionDoneMsg{out: "", err: withStep("校验封面", fmt.Errorf("版本 %d (%s) 缺少封面，封面为必填项", i+1, v.version))}
					return
				}
			}

			apiKey, err := lib.NewSfFolder().GetKey()
			if err != nil || apiKey == "" {
				ch <- actionDoneMsg{out: "", err: withStep("上传/鉴权", fmt.Errorf("未登录或缺少 API Key，请先登录"))}
				return
			}
			client := lib.NewClient(meta.DefaultDomain, apiKey)

			mvList := make([]*lib.ModelVersion, len(versions))
			var wg sync.WaitGroup
			sem := make(chan struct{}, 3)
			var mu sync.Mutex
			var anyCanceled bool
			var errs []error

			for i, v := range versions {
				wg.Add(1)
				idx := i
				ver := v
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					fileIndex := fmt.Sprintf("%d/%d", idx+1, len(versions))

					verVersion := strings.TrimSpace(ver.version)
					if verVersion == "" {
						mu.Lock()
						errs = append(errs, withStep("上传/校验版本号", fmt.Errorf("版本[%d] 的版本号为空，请检查输入", idx+1)))
						mu.Unlock()
						return
					}

					// 使用统一的封面上传逻辑
					var coverUrls []string
					if strings.TrimSpace(ver.cover) != "" {
						coverUrl, cerr := lib.UploadCover(client, ver.cover, ctx)
						if cerr != nil {
							// 检查是否是用户取消
							if errors.Is(cerr, context.Canceled) {
								mu.Lock()
								anyCanceled = true
								mu.Unlock()
								return
							}
							mu.Lock()
							errs = append(errs, cerr)
							mu.Unlock()
							return
						}
						if coverUrl != "" {
							coverUrls = append(coverUrls, coverUrl)
						}
					}

					// 使用统一的上传逻辑
					st, err := os.Stat(ver.path)
					if err != nil {
						mu.Lock()
						errs = append(errs, withStep("上传/读取文件信息", err))
						mu.Unlock()
						return
					}

					relPath, err := filepath.Rel(filepath.Dir(ver.path), ver.path)
					if err != nil {
						relPath = filepath.Base(ver.path)
					}
					f := &lib.FileToUpload{
						Path:    filepath.ToSlash(ver.path),
						RelPath: filepath.ToSlash(relPath),
						Size:    st.Size(),
					}

					// 使用 UnifiedUpload 进行上传
					_, err = lib.UnifiedUpload(lib.UploadOptions{
						File:      f,
						Client:    client,
						ModelType: u.typ,
						Context:   ctx,
						FileIndex: fileIndex,
						ProgressFunc: func(consumed, total int64) {
							select {
							case ch <- uploadProgMsg{
								fileIndex: fileIndex,
								fileName:  filepath.Base(f.RelPath),
								consumed:  consumed,
								total:     total,
								verIdx:    idx,
							}:
							default:
							}
						},
					})
					if err != nil {
						// 检查是否是用户取消
						if errors.Is(err, context.Canceled) {
							mu.Lock()
							anyCanceled = true
							mu.Unlock()
							return
						}
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}

					mv := &lib.ModelVersion{
						Version:      verVersion,
						BaseModel:    ver.base,
						Introduction: ver.intro,
						Public:       false,
						Sign:         f.Signature,
						Path:         ver.path,
					}
					if len(coverUrls) > 0 {
						mv.CoverUrls = coverUrls
					}
					mvList[idx] = mv
				}()
			}

			wg.Wait()

			if anyCanceled {
				var sb strings.Builder
				sb.WriteString("✓ 上传已取消\n\n")
				sb.WriteString("已上传的部分已保存checkpoint，下次上传相同文件时会自动续传。\n\n")
				folder, _ := lib.GetCheckpointDir()
				if folder != "" {
					sb.WriteString(fmt.Sprintf("Checkpoint文件位置: %s\n", folder))
				}
				ch <- actionDoneMsg{out: sb.String(), err: nil}
				return
			}

			finalVersions := make([]*lib.ModelVersion, 0, len(mvList))
			for _, mv := range mvList {
				if mv != nil {
					finalVersions = append(finalVersions, mv)
				}
			}
			if len(finalVersions) == 0 {
				var sb strings.Builder
				sb.WriteString("所有版本上传失败\n")
				for _, e := range errs {
					sb.WriteString("- ")
					sb.WriteString(e.Error())
					sb.WriteString("\n")
				}
				ch <- actionDoneMsg{out: sb.String(), err: withStep("上传/全部失败", fmt.Errorf("上传失败"))}
				return
			}

			if _, err := client.CommitModelV2(u.name, u.typ, finalVersions); err != nil {
				ch <- actionDoneMsg{err: withStep("上传/发布模型", err)}
				return
			}

			var out strings.Builder
			out.WriteString("Uploaded successfully\n")
			if len(finalVersions) != len(versions) {
				out.WriteString(fmt.Sprintf("部分版本失败：成功 %d / %d\n", len(finalVersions), len(versions)))
				for _, e := range errs {
					out.WriteString("- ")
					out.WriteString(e.Error())
					out.WriteString("\n")
				}
			}
			ch <- actionDoneMsg{out: out.String(), err: nil}
		}()
		return uploadStartMsg{ch: ch, cancel: cancel}
	}
}
