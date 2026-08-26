package cmd

import (
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
)

// ResolveClient 从 args 构建 API 客户端：apiKey 空则从本地读取并回写 args。
func ResolveClient(args *config.Argument) (string, *lib.Client, error) {
	apiKey := args.ApiKey
	if apiKey == "" {
		var err error
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return "", nil, err
		}
	}
	args.ApiKey = apiKey
	client := lib.NewClient(args.BaseDomain, apiKey)
	return apiKey, client, nil
}
