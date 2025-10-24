#!/usr/bin/env python3
"""
本地生成 manifest.json 工具（用于测试）

用法：
  make build_release VERSION=v0.0.2
  python3 tools/generate_manifest.py
"""

import os
import json
from pathlib import Path
from datetime import datetime


def main():
    version = os.getenv("VERSION", "v0.0.1")
    dist_dir = Path("dist")
    
    if not dist_dir.exists():
        print("❌ dist 目录不存在，请先运行 make build_release")
        return
    
    platforms = {}
    
    # 遍历所有二进制文件
    for binary_file in sorted(dist_dir.glob("bizyair-*")):
        if binary_file.suffix == ".sha256":
            continue
        
        filename = binary_file.name
        
        # 读取 SHA256
        sha256_file = binary_file.with_suffix(binary_file.suffix + ".sha256")
        if sha256_file.exists():
            checksum = sha256_file.read_text().strip()
        else:
            checksum = "未计算"
        
        # 解析平台信息
        parts = filename.replace(".exe", "").split("-")
        if len(parts) >= 4:
            platform_os = parts[-2]
            platform_arch = parts[-1]
            platform_key = f"{platform_os}-{platform_arch}"
        else:
            continue
        
        # 构建 URL（实际部署时会替换）
        url = f"https://storage.bizyair.cn/releases/{version}/{platform_key}/{filename}"
        
        platforms[platform_key] = {
            "filename": filename,
            "url": url,
            "checksum": f"sha256:{checksum}"
        }
    
    # 生成 manifest
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
    
    # 保存
    manifest_path = dist_dir / "manifest.json"
    with open(manifest_path, 'w', encoding='utf-8') as f:
        json.dump(manifest, f, indent=2, ensure_ascii=False)
    
    print(f"✅ Manifest 已生成: {manifest_path}")
    print(f"版本: {version}")
    print(f"平台数: {len(platforms)}")


if __name__ == "__main__":
    main()

