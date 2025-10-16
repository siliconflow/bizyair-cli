<h1 align="center">BizyAir CLI</h1>

<p align="center">
<p>

BizyAir CLI 是一个开源命令行工具，您可以从 [GitHub](https://github.com/siliconflow/bizyair-cli) 获取最新版本。

## 简介

BizyAir CLI 是用于管理 BizyAir 上文件的命令行工具。它提供了一种简便的方式来上传和管理您的 BizyAir 文件。

## 发布版本

所有发布版本请[点击这里查看](https://github.com/siliconflow/bizyair-cli/releases)。

## 安装

BizyAir CLI 支持 Linux、macOS 和 Windows 平台。
Linux、Windows 和 Mac 的二进制文件以 tarball 形式提供在[发布页面](https://github.com/siliconflow/bizyair-cli/releases)。

- **Linux 安装**

  ```shell
  VERSION=0.1.0
  tar -xzvf bizyair-cli-linux-$VERSION-amd64.tar.gz
  install bizyair /usr/local/bin
  ```

- **通过 GO 安装**

  ```shell
  # 注意：这将安装开发版本！
  go install github.com/siliconflow/bizyair-cli@latest
  ```

## 从源码构建

BizyAir CLI 当前使用 GO v1.22.X 或更高版本。
要从源码构建，您需要：

1. 克隆仓库
2. 构建并运行可执行文件

   ```shell
   make build && ./execs/bizyair
   ```

- Windows 系统运行：`make build_windows`
- MacOS 系统运行：`make build_mac`
- Linux 系统运行：`make build_linux` 或 `make build_linux_arm64`

---

## 快速开始

### 登录

BizyAir CLI 使用 API Key 进行身份验证。要登录您的设备，运行以下命令：

```bash
# 如果您已设置环境变量 SF_API_KEY
bizyair login

# 或使用 --key/-k 选项指定
bizyair login -k $SF_API_KEY
```

### 登出

要登出您的设备，运行以下命令：

```bash
bizyair logout
```

### 上传文件

要上传文件到 BizyAir，运行以下命令：

```bash
bizyair upload -n mymodel -t [LoRA | Controlnet | Checkpoint] -p /local/path/file -b [basemodel] -cover /path/to/cover.jpg
```

**新功能特性：**

- **自动断点续传**：上传中断后，重新运行相同命令会自动从断点继续
- **并发上传**：默认使用 3 个并发上传，显著提升速度
- **分片上传**：大文件（>100MB）自动使用分片上传
- **封面上传**：支持本地文件和 URL，自动上传到 OSS（**必填项**）

您可以使用 `-n`、`-t`、`-b`、`-p` 和 `-cover` 标志分别指定模型名称、模型类型、基础模型、上传路径和封面（必填）。

#### 封面图片/视频上传（必填）

`-cover` 标志为**必填项**，同时支持 URL 和本地文件：

```bash
# 使用本地封面文件（自动上传到 OSS）
bizyair upload -n mymodel -t Checkpoint \
  -p /local/path/model.safetensors -b SDXL \
  -cover "/local/path/cover.jpg"

# 使用远程 URL（自动下载并上传到 OSS）
bizyair upload -n mymodel -t Checkpoint \
  -p /local/path/model.safetensors -b SDXL \
  -cover "https://example.com/cover.jpg"
```

**支持的封面格式：** `.jpg`、`.jpeg`、`.png`、`.gif`、`.webp`、`.mp4`、`.webm`、`.mov`  
**视频大小限制：** 100MB  
**注意：** 每个上传的版本都必须提供封面。

#### 多版本上传

您可以使用 `-v`、`-i`、`-cover` 标志指定版本名称、介绍和封面。列表中的每个值对应特定版本：

```bash
bizyair upload -n mymodel -t Checkpoint \
-v "v1" -b SDXL -p /local/path/file1 -i "sdxl checkpoint1" -cover "/path/cover1.jpg" \
-v "v2" -b SD1.5 -p /local/path/file2 -i "sd1.5 checkpoint" -cover "https://example.com/cover2.jpg" \
-v "v3" -b SDXL -p /local/path/file3 -i "another checkpoint" -cover "/path/cover3.png"
```

**注意：**

- 如果未指定 `-v`，将使用默认版本名称。
- **每个版本都必须提供封面**，封面数量必须与文件数量相同。

#### 断点续传

如果上传被中断（例如按 Ctrl+C），只需再次运行相同的命令。CLI 将自动从中断的地方继续：

```bash
# 第一次上传（中断）
bizyair upload -n mymodel -t Checkpoint -p /path/large-model.safetensors -b SDXL -cover "/path/cover.jpg"
# ^C (中断)

# 重新运行相同命令，自动续传
bizyair upload -n mymodel -t Checkpoint -p /path/large-model.safetensors -b SDXL -cover "/path/cover.jpg"
# ✓ 从断点继续上传
```

### 查看模型列表

要查看您在云端的所有模型，运行以下命令：

```bash
bizyair model ls -t Checkpoint
```

您必须使用 `-t` 标志指定模型类型。

您可以使用 `--public` 标志查看所有公开模型。默认情况下只显示您的私有模型。

### 查看模型文件

要查看模型中的所有文件，运行以下命令：

```bash
bizyair model ls-files -n mymodel -t Checkpoint
```

您可以使用 `--public` 标志查看所有公开模型文件。默认情况下只显示您的私有模型文件。

如果您想以树形视图查看模型中的文件，运行以下命令：

```bash
bizyair model ls-files -n mymodel -t Checkpoint --tree
```

您必须使用 `-n` 和 `-t` 标志分别指定模型名称和模型类型。

### 删除模型

要从云端删除模型，运行以下命令：

```bash
bizyair model rm -n mymodel -t Checkpoint
```

您必须使用 `-n` 和 `-t` 标志分别指定模型名称和模型类型。
