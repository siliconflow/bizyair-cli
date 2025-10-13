package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/filehash"
	"github.com/siliconflow/bizyair-cli/lib/format"
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

					var coverUrls []string
					if strings.TrimSpace(ver.cover) != "" {
						item := strings.TrimSpace(ver.cover)
						if i := strings.Index(item, ";"); i >= 0 {
							item = strings.TrimSpace(item[:i])
						}
						localPath := item
						cleanup1 := func() {}
						if lib.IsHTTPURL(item) {
							if p, cfn, derr := lib.DownloadToTemp(item); derr == nil {
								localPath = p
								cleanup1 = cfn
							} else {
								mu.Lock()
								errs = append(errs, withStep("上传/下载封面", fmt.Errorf("封面下载失败: %s, %v", item, derr)))
								mu.Unlock()
								cleanup1()
								return
							}
						}
						tkn, terr := client.GetUploadToken(filepath.Base(localPath), "inputs")
						if terr != nil {
							cleanup1()
							mu.Lock()
							errs = append(errs, withStep("上传/封面凭证", fmt.Errorf("封面获取上传凭证失败: %s, %v", item, terr)))
							mu.Unlock()
							return
						}
						fileRec := tkn.Data.File
						sto := tkn.Data.Storage
						ossCli, oerr := lib.NewAliOssStorageClient(sto.Endpoint, sto.Bucket, fileRec.AccessKeyId, fileRec.AccessKeySecret, fileRec.SecurityToken)
						if oerr != nil {
							cleanup1()
							mu.Lock()
							errs = append(errs, withStep("上传/封面OSS客户端", fmt.Errorf("封面创建 OSS 客户端失败: %s, %v", item, oerr)))
							mu.Unlock()
							return
						}
						coverFile := &lib.FileToUpload{Path: localPath, RelPath: filepath.Base(localPath), Size: 0}
						if _, uerr := ossCli.UploadFileCtx(ctx, coverFile, fileRec.ObjectKey, fileIndex, nil); uerr != nil {
							cleanup1()
							mu.Lock()
							errs = append(errs, withStep("上传/封面上传", fmt.Errorf("封面上传 OSS 失败: %s, %v", item, uerr)))
							mu.Unlock()
							return
						}
						if commit, cerr := client.CommitInputResource(filepath.Base(localPath), fileRec.ObjectKey); cerr != nil {
							cleanup1()
							mu.Lock()
							errs = append(errs, withStep("上传/封面提交", fmt.Errorf("封面提交失败: %s, %v", item, cerr)))
							mu.Unlock()
							return
						} else if commit != nil && commit.Data.Url != "" {
							coverUrls = append(coverUrls, commit.Data.Url)
						}
						cleanup1()
					}

					st, err := os.Stat(ver.path)
					if err != nil {
						mu.Lock()
						errs = append(errs, withStep("上传/读取文件信息", err))
						mu.Unlock()
						return
					}
					if st.IsDir() {
						mu.Lock()
						errs = append(errs, withStep("上传/校验路径", fmt.Errorf("仅支持文件上传: %s", ver.path)))
						mu.Unlock()
						return
					}

					relPath, err := filepath.Rel(filepath.Dir(ver.path), ver.path)
					if err != nil {
						mu.Lock()
						errs = append(errs, withStep("上传/计算哈希", err))
						mu.Unlock()
						return
					}
					f := &lib.FileToUpload{Path: filepath.ToSlash(ver.path), RelPath: filepath.ToSlash(relPath), Size: st.Size()}

					sha256sum, md5Hash, err := filehash.CalculateHash(f.Path)
					if err != nil {
						mu.Lock()
						errs = append(errs, err)
						mu.Unlock()
						return
					}
					f.Signature = sha256sum

					// 先按 sha256 尝试本地 checkpoint 续传
					resumed := false
					if cpPath, gerr := lib.GetCheckpointFile(sha256sum); gerr == nil {
						if cp, lerr := lib.LoadCheckpoint(cpPath); lerr == nil && cp != nil {
							if lib.ValidateCheckpoint(cp, f) {
								var ossClient *lib.AliOssStorageClient
								useCreds := cp.AccessKeyId != "" && cp.AccessKeySecret != ""
								if useCreds && strings.TrimSpace(cp.Expiration) != "" {
									if exp, perr := time.Parse(time.RFC3339, cp.Expiration); perr == nil {
										if time.Now().After(exp) {
											useCreds = false
										}
									}
								}
								if useCreds {
									if cli, oerr := lib.NewAliOssStorageClient(cp.Endpoint, cp.Bucket, cp.AccessKeyId, cp.AccessKeySecret, cp.SecurityToken); oerr == nil {
										cli.SetExpiration(cp.Expiration)
										ossClient = cli
									}
								}

								// 凭证不可用或过期则刷新凭证
								if ossClient == nil {
									ossCert, err := client.OssSign(sha256sum, u.typ)
									if err != nil {
										mu.Lock()
										errs = append(errs, err)
										mu.Unlock()
										return
									}
									fileRecord := ossCert.Data.File
									storage := ossCert.Data.Storage
									if cli, oerr := lib.NewAliOssStorageClient(storage.Endpoint, storage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken); oerr == nil {
										cli.SetExpiration(fileRecord.Expiration)
										ossClient = cli
									} else {
										mu.Lock()
										errs = append(errs, withStep("上传/获取上传签名", oerr))
										mu.Unlock()
										return
									}
								}

								// 续传：使用 checkpoint 的 objectKey
								_, err = ossClient.UploadFileMultipart(ctx, f, cp.ObjectKey, fileIndex, func(consumed, total int64) {
									// 显示进度（总量>0时）
									if total > 0 {
										_ = format.FormatBytes(consumed)
										_ = format.FormatBytes(total)
									}
									select {
									case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: consumed, total: total, verIdx: idx}:
									default:
									}
								})
								if err != nil {
									if errors.Is(err, context.Canceled) {
										mu.Lock()
										anyCanceled = true
										mu.Unlock()
										return
									}
									mu.Lock()
									errs = append(errs, withStep("上传/OSS上传模型文件", err))
									mu.Unlock()
									return
								}
								// 提交使用续传时的 key
								commitKey := cp.ObjectKey
								if f.RemoteKey != "" {
									commitKey = f.RemoteKey
								}
								if _, err = client.CommitFileV2(f.Signature, commitKey, md5Hash, u.typ); err != nil {
									mu.Lock()
									errs = append(errs, withStep("上传/提交模型文件", err))
									mu.Unlock()
									return
								}
								resumed = true
							}
						}
					}

					if !resumed {
						ossCert, err := client.OssSign(sha256sum, u.typ)
						if err != nil {
							mu.Lock()
							errs = append(errs, err)
							mu.Unlock()
							return
						}
						fileRecord := ossCert.Data.File

						if fileRecord.Id == 0 {
							storage := ossCert.Data.Storage
							ossClient, err := lib.NewAliOssStorageClient(storage.Endpoint, storage.Bucket, fileRecord.AccessKeyId, fileRecord.AccessKeySecret, fileRecord.SecurityToken)
							if err != nil {
								mu.Lock()
								errs = append(errs, withStep("上传/获取上传签名", err))
								mu.Unlock()
								return
							}
							ossClient.SetExpiration(fileRecord.Expiration)
							_, err = ossClient.UploadFileMultipart(ctx, f, fileRecord.ObjectKey, fileIndex, func(consumed, total int64) {
								// 显示进度（总量>0时）
								if total > 0 {
									_ = format.FormatBytes(consumed)
									_ = format.FormatBytes(total)
								}
								select {
								case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: consumed, total: total, verIdx: idx}:
								default:
								}
							})
							if err != nil {
								if errors.Is(err, context.Canceled) {
									mu.Lock()
									anyCanceled = true
									mu.Unlock()
									return
								}
								mu.Lock()
								errs = append(errs, withStep("上传/OSS上传模型文件", err))
								mu.Unlock()
								return
							}
							commitKey := fileRecord.ObjectKey
							if f.RemoteKey != "" {
								commitKey = f.RemoteKey
							}
							if _, err = client.CommitFileV2(f.Signature, commitKey, md5Hash, u.typ); err != nil {
								mu.Lock()
								errs = append(errs, withStep("上传/提交模型文件", err))
								mu.Unlock()
								return
							}
						} else {
							f.Id = fileRecord.Id
							f.RemoteKey = fileRecord.ObjectKey
							select {
							case ch <- uploadProgMsg{fileIndex: fileIndex, fileName: filepath.Base(f.RelPath), consumed: f.Size, total: f.Size, verIdx: idx}:
							default:
							}
						}
					}

					mv := &lib.ModelVersion{Version: verVersion, BaseModel: ver.base, Introduction: ver.intro, Public: false, Sign: f.Signature, Path: ver.path}
					if len(coverUrls) > 0 {
						mv.CoverUrls = coverUrls
					}
					mvList[idx] = mv
				}()
			}

			wg.Wait()

			if anyCanceled {
				ch <- actionDoneMsg{out: "上传已取消\n", err: nil}
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
