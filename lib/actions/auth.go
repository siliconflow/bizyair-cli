package actions

import (
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// ExecuteLogin 执行登录操作
// 验证API Key并保存到本地
func ExecuteLogin(apiKey string) LoginResult {
	if apiKey == "" {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep("登录", lib.NewValidationError("API Key不能为空")),
		}
	}

	// 1. 验证API Key
	client := lib.NewClient(meta.AuthDomain, apiKey)
	_, err := client.UserInfo()
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep("登录校验", err),
		}
	}

	// 2. 保存API Key到本地
	err = lib.NewSfFolder().SaveKey(apiKey)
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep("保存凭据", err),
		}
	}

	return LoginResult{
		Success: true,
		ApiKey:  apiKey,
	}
}

// ExecuteLogout 执行登出操作
// 删除本地保存的API Key
func ExecuteLogout() error {
	err := lib.NewSfFolder().RemoveKey()
	if err != nil {
		return lib.WithStep("登出", err)
	}
	return nil
}

// ExecuteWhoami 执行whoami操作
// 查询当前登录用户的信息
func ExecuteWhoami(apiKey string) WhoamiResult {
	if apiKey == "" {
		return WhoamiResult{
			Error: lib.WithStep("whoami", lib.NewValidationError("未登录或缺少API Key")),
		}
	}

	client := lib.NewClient(meta.AuthDomain, apiKey)
	info, err := client.UserInfo()
	if err != nil {
		return WhoamiResult{
			Error: lib.WithStep("查询用户信息", err),
		}
	}

	return WhoamiResult{
		Name:  info.Data.Name,
		Email: info.Data.Email,
	}
}
