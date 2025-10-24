#!/bin/bash
# BizyAir CLI 发布上传脚本（Bash 版本）
# 
# 注意：推荐使用 upload_release.py（Python 版本），此脚本仅作为备用

set -e

VERSION=${VERSION:-}
API_KEY=${API_KEY:-}
BASE_DOMAIN=${BASE_DOMAIN:-https://api.bizyair.cn}

if [ -z "$VERSION" ] || [ -z "$API_KEY" ]; then
    echo "❌ 错误: 缺少必要的环境变量 VERSION 或 API_KEY"
    exit 1
fi

echo "开始上传版本: $VERSION"
echo "API Domain: $BASE_DOMAIN"

# 检查 dist 目录
if [ ! -d "dist" ]; then
    echo "❌ 错误: dist 目录不存在"
    exit 1
fi

# 检查必要工具
if ! command -v curl &> /dev/null; then
    echo "❌ 错误: 需要 curl 工具"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "❌ 错误: 需要 jq 工具"
    exit 1
fi

# 函数：获取上传 token
get_upload_token() {
    local filename=$1
    local url="${BASE_DOMAIN}/x/v1/upload/token?file_name=${filename}&file_type=cli&ignore_date=true&final_file_name=true"
    
    local response=$(curl -s -H "Authorization: Bearer ${API_KEY}" "$url")
    local code=$(echo "$response" | jq -r '.code')
    
    if [ "$code" != "20000" ]; then
        echo "❌ 获取 token 失败: $(echo $response | jq -r '.message')"
        exit 1
    fi
    
    echo "$response"
}

# 函数：上传文件到 OSS（需要 ossutil 或 s3cmd）
upload_to_oss() {
    local file=$1
    local token_data=$2
    
    # 这里需要使用 OSS SDK 或工具上传
    # 由于 Bash 实现 OSS 签名较复杂，建议使用 Python 版本
    echo "⚠️  Bash 版本不支持直接上传到 OSS"
    echo "请使用 Python 版本: tools/upload_release.py"
    exit 1
}

echo "⚠️  此脚本为备用版本，推荐使用 Python 版本"
echo "运行: python tools/upload_release.py"

