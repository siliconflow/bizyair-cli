#!/usr/bin/env python3
"""
BizyAir CLI 发布上传脚本

功能：
1. 从环境变量获取版本号、API Key
2. 遍历 dist 目录下的所有二进制文件
3. 为每个文件获取上传 token
4. 上传到 OSS
5. 生成并上传 manifest.json
"""

import os
import sys
import json
import requests
from datetime import datetime
from pathlib import Path
from alibabacloud_oss_v2 import models as oss_models
from alibabacloud_oss_v2.client import Client as OSSClient
from alibabacloud_credentials.client import Client as CredClient
from alibabacloud_credentials import models as credential_models


def get_upload_token(api_key, base_domain, filename):
    """获取 CLI 上传 token"""
    url = f"{base_domain}/x/v1/upload/token"
    params = {
        "file_name": filename,
        "file_type": "cli",
        "ignore_date": "true",
        "final_file_name": "true"
    }
    headers = {
        "Authorization": f"Bearer {api_key}"
    }
    
    response = requests.get(url, params=params, headers=headers)
    response.raise_for_status()
    
    data = response.json()
    if data.get("code") != 20000:
        raise Exception(f"API 错误: {data.get('message')}")
    
    return data["data"]


def upload_to_oss(file_path, token_data):
    """上传文件到 OSS"""
    file_info = token_data["file"]
    storage_info = token_data["storage"]
    
    # 创建 OSS 客户端
    config = oss_models.Config(
        credentials_provider=credential_models.StaticCredentialProvider(
            access_key_id=file_info["access_key_id"],
            access_key_secret=file_info["access_key_secret"],
            security_token=file_info.get("security_token")
        ),
        region=storage_info["region"]
    )
    
    client = OSSClient(config)
    
    # 上传文件
    with open(file_path, 'rb') as f:
        request = oss_models.PutObjectRequest(
            bucket=storage_info["bucket"],
            key=file_info["object_key"],
            body=f
        )
        client.put_object(request)
    
    # 返回完整 URL
    url = f"https://storage.bizyair.cn/{file_info['object_key']}"
    return url, file_info["object_key"]


def upload_manifest(manifest_data, api_key, base_domain):
    """上传 manifest.json"""
    # 将 manifest 转为 JSON 字符串
    manifest_json = json.dumps(manifest_data, indent=2, ensure_ascii=False)
    
    # 获取上传 token
    token_data = get_upload_token(api_key, base_domain, "manifest.json")
    
    file_info = token_data["file"]
    storage_info = token_data["storage"]
    
    # 创建 OSS 客户端
    config = oss_models.Config(
        credentials_provider=credential_models.StaticCredentialProvider(
            access_key_id=file_info["access_key_id"],
            access_key_secret=file_info["access_key_secret"],
            security_token=file_info.get("security_token")
        ),
        region=storage_info["region"]
    )
    
    client = OSSClient(config)
    
    # 上传 manifest
    request = oss_models.PutObjectRequest(
        bucket=storage_info["bucket"],
        key=file_info["object_key"],
        body=manifest_json.encode('utf-8')
    )
    client.put_object(request)
    
    url = f"https://storage.bizyair.cn/{file_info['object_key']}"
    print(f"✅ Manifest 已上传: {url}")
    return url


def main():
    # 从环境变量获取配置
    version = os.getenv("VERSION")
    api_key = os.getenv("API_KEY")
    base_domain = os.getenv("BASE_DOMAIN", "https://api.bizyair.cn")
    
    if not version or not api_key:
        print("❌ 错误: 缺少必要的环境变量 VERSION 或 API_KEY")
        sys.exit(1)
    
    print(f"开始上传版本: {version}")
    print(f"API Domain: {base_domain}")
    
    # 读取 dist 目录
    dist_dir = Path("dist")
    if not dist_dir.exists():
        print("❌ 错误: dist 目录不存在")
        sys.exit(1)
    
    # 收集所有二进制文件
    binary_files = [f for f in dist_dir.glob("bizyair-*") if not f.suffix == ".sha256"]
    
    if not binary_files:
        print("❌ 错误: 未找到二进制文件")
        sys.exit(1)
    
    print(f"找到 {len(binary_files)} 个文件待上传")
    
    # 准备 manifest 数据
    platforms = {}
    
    for binary_file in binary_files:
        filename = binary_file.name
        print(f"\n处理文件: {filename}")
        
        # 读取 SHA256
        sha256_file = binary_file.with_suffix(binary_file.suffix + ".sha256")
        if sha256_file.exists():
            checksum = sha256_file.read_text().strip()
        else:
            print(f"  ⚠️  警告: 未找到 SHA256 文件")
            checksum = ""
        
        # 确定平台
        # 文件名格式: bizyair-v0.0.1-macos-amd64
        parts = filename.replace(".exe", "").split("-")
        if len(parts) >= 4:
            platform_os = parts[-2]  # macos, linux, windows
            platform_arch = parts[-1]  # amd64, arm64
            platform_key = f"{platform_os}-{platform_arch}"
        else:
            print(f"  ⚠️  警告: 无法解析平台信息，跳过")
            continue
        
        print(f"  平台: {platform_key}")
        print(f"  SHA256: {checksum}")
        
        # 获取上传 token
        print(f"  获取上传凭证...")
        try:
            token_data = get_upload_token(api_key, base_domain, filename)
        except Exception as e:
            print(f"  ❌ 获取 token 失败: {e}")
            sys.exit(1)
        
        # 上传文件
        print(f"  上传文件...")
        try:
            url, object_key = upload_to_oss(str(binary_file), token_data)
            print(f"  ✅ 上传成功: {url}")
        except Exception as e:
            print(f"  ❌ 上传失败: {e}")
            sys.exit(1)
        
        # 添加到 platforms
        platforms[platform_key] = {
            "filename": filename,
            "url": url,
            "checksum": f"sha256:{checksum}"
        }
    
    # 生成 manifest
    print("\n生成 manifest.json...")
    manifest = {
        "latest_version": version,
        "releases": {
            version: {
                "version": version,
                "release_date": datetime.utcnow().isoformat() + "Z",
                "platforms": platforms
            }
        }
    }
    
    # 保存到本地（用于调试）
    manifest_path = dist_dir / "manifest.json"
    with open(manifest_path, 'w', encoding='utf-8') as f:
        json.dump(manifest, f, indent=2, ensure_ascii=False)
    print(f"Manifest 已保存到: {manifest_path}")
    
    # 上传 manifest
    print("\n上传 manifest.json...")
    try:
        upload_manifest(manifest, api_key, base_domain)
    except Exception as e:
        print(f"❌ 上传 manifest 失败: {e}")
        sys.exit(1)
    
    print(f"\n✅ 所有文件已成功上传！版本: {version}")


if __name__ == "__main__":
    main()

