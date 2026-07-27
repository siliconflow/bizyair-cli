package actions

import (
	"context"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// ExecuteLogin 执行登录操作
// 验证API Key并保存到本地
func ExecuteLogin(api lib.BizyAPI, apiKey string) LoginResult {
	return ExecuteLoginContext(context.Background(), api, apiKey)
}

func ExecuteLoginContext(ctx context.Context, api lib.BizyAPI, apiKey string) LoginResult {
	if apiKey == "" {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep(i18n.T("step.login"), i18n.NewError("validation.api_key_required", nil, nil)),
		}
	}

	_, err := api.UserInfoContext(ctx)
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep(i18n.T("step.login_validation"), err),
		}
	}

	err = lib.NewSfFolder().SaveKey(apiKey)
	if err != nil {
		return LoginResult{
			Success: false,
			Error:   lib.WithStep(i18n.T("step.save_credentials"), err),
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
		return lib.WithStep(i18n.T("step.logout"), err)
	}
	return nil
}
