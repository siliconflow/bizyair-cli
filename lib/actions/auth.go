package actions

import (
	"github.com/siliconflow/bizyair-cli/lib"
)

// ExecuteLogin 执行登录操作
// 验证API Key并保存到本地
func ExecuteLogin(api lib.BizyAPI, apiKey string) LoginResult {
	if apiKey == "" {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep("登录", lib.NewValidationError("API Key不能为空")),
		}
	}

	_, err := api.UserInfo()
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep("登录校验", err),
		}
	}

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
