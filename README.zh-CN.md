<h1 align="center">BizyAir CLI</h1>

<p align="center"><strong>简体中文</strong> | <a href="./README.md">English</a></p>

<p align="center">
<p>

BizyAir CLI 是一个开源命令行工具，您可以从 [GitHub](https://github.com/siliconflow/bizyair-cli) 获取最新版本。

## 简介

BizyAir CLI 是用于管理 BizyAir 上模型文件的命令行工具。它提供了**交互式界面（TUI）**和**命令行（CLI）**两种使用模式，让您可以轻松上传和管理 BizyAir 模型。

### ✨ 核心特性

- 🎨 **交互式 TUI 界面** - 友好的图形化交互体验，无需记忆命令
- 🚀 **断点续传** - 上传中断后自动从断点继续，节省时间
- ⚡ **并发上传** - 最多 3 个并发，显著提升上传速度
- 📦 **分片上传** - 大文件（>100MB）自动分片上传，更稳定
- 🖼️ **智能封面处理** - 自动转换为 WebP 格式，支持图片和视频
- 📝 **灵活的介绍输入** - 支持直接输入或从文件导入（.txt/.md）
- 📋 **YAML 批量上传** - 通过配置文件一次上传多个模型
- 📊 **实时进度显示** - 上传速率、进度条实时更新
- 🔄 **自动升级** - 一键升级到最新版本，支持版本检查和安全回滚

## 发布版本

所有发布版本请[点击这里查看](https://github.com/siliconflow/bizyair-cli/releases)。

## 安装

BizyAir CLI 支持 Linux、macOS 和 Windows 平台。所有平台的二进制文件都可以从 [GitHub Release 页面](https://github.com/siliconflow/bizyair-cli/releases) 下载。

### Linux 安装

1. 从 [Release 页面](https://github.com/siliconflow/bizyair-cli/releases) 下载最新版本：

   ```bash
   # 设置版本号（请替换为最新版本）
   VERSION=0.2.1

   # 下载 Linux amd64 版本
   wget https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-linux-amd64.tar.gz

   # 或下载 Linux arm64 版本
   # wget https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-linux-arm64.tar.gz
   ```

2. 解压并安装：

   ```bash
   # 解压文件
   tar -xzvf bizyair-v${VERSION}-linux-amd64.tar.gz

   # 安装到系统路径
   sudo install bizyair /usr/local/bin/bizyair

   # 或者安装到用户目录（无需 sudo）
   # mkdir -p ~/.local/bin
   # install bizyair ~/.local/bin/bizyair
   # 确保 ~/.local/bin 在您的 PATH 中
   ```

3. 验证安装：

   ```bash
   bizyair --version
   ```

### macOS 安装

1. 从 [Release 页面](https://github.com/siliconflow/bizyair-cli/releases) 下载最新版本：

   ```bash
   # 设置版本号（请替换为最新版本）
   VERSION=0.2.1

   # 下载 macOS arm64 版本（Apple Silicon，如 M1/M2/M3）
   curl -LO https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-macos-arm64.tar.gz

   # 或下载 macOS amd64 版本（Intel 芯片）
   # curl -LO https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-macos-amd64.tar.gz
   ```

2. 解压并安装：

   ```bash
   # 解压文件
   tar -xzvf bizyair-v${VERSION}-macos-arm64.tar.gz

   # 添加执行权限
   chmod +x bizyair

   # 安装到系统路径
   sudo install bizyair /usr/local/bin/bizyair

   # 或者安装到用户目录（无需 sudo）
   # mkdir -p ~/.local/bin
   # install bizyair ~/.local/bin/bizyair
   # 确保 ~/.local/bin 在您的 PATH 中
   ```

3. 首次运行时，macOS 可能会提示安全警告，请按以下步骤操作：

   ```bash
   # 移除隔离属性
   sudo xattr -d com.apple.quarantine /usr/local/bin/bizyair
   ```

   或者在系统设置中允许运行：

   - 打开 **系统设置** → **隐私与安全性**
   - 找到被阻止的 `bizyair` 并点击 **仍要打开**

4. 验证安装：

   ```bash
   bizyair --version
   ```

### Windows 安装

1. 从 [Release 页面](https://github.com/siliconflow/bizyair-cli/releases) 下载最新版本：

   - 下载 `bizyair-v{VERSION}-windows-amd64.zip` 文件

2. 解压文件：

   - 右键点击下载的 `.zip` 文件
   - 选择 **解压全部** 或使用解压工具（如 7-Zip）
   - 将 `bizyair.exe` 解压到您选择的目录（例如 `C:\Program Files\BizyAir\`）

3. 添加到系统 PATH（推荐）：

   - 右键点击 **此电脑** → **属性** → **高级系统设置**
   - 点击 **环境变量**
   - 在 **系统变量** 中找到 `Path`，点击 **编辑**
   - 点击 **新建**，添加 `bizyair.exe` 所在的目录路径
   - 点击 **确定** 保存

4. 验证安装：

   打开 **命令提示符** 或 **PowerShell**：

   ```powershell
   bizyair --version
   ```

   如果没有添加到 PATH，可以使用完整路径运行：

   ```powershell
   C:\Program Files\BizyAir\bizyair.exe --version
   ```

## 从源码构建

BizyAir CLI 当前使用 Go 1.24 或更高版本。
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

## 使用模式

BizyAir CLI 提供两种使用模式，您可以根据自己的喜好选择：

### 🌐 界面语言

BizyAir CLI 标准支持英语（`en`）和简体中文（`zh-CN`）。语言在每次启动时确定一次，不会写入用户配置，也不能在 TUI 内切换。

语言选择优先级为：`--lang` > 系统语言 > 英语。未指定 `--lang` 时，简体中文和繁体中文系统都会使用简体中文界面；其他、空值或无法识别的系统语言统一使用英语。

```bash
bizyair --lang zh-CN upload --help
bizyair upgrade --lang en --check
```

`--lang` 可以放在子命令之前或之后，且只接受 `en` 和 `zh-CN`。系统 locale 中的 `zh-CN`、`zh-Hans`、`zh-TW`、`zh-Hant` 等会被自动识别，并统一选择简体中文界面。显式设置为其他值会返回错误并列出支持值。

### 🎨 交互式界面（TUI）- 推荐新手使用

直接运行 `bizyair` 命令即可进入友好的交互式界面：

```bash
bizyair
```

TUI 模式特点：

- ✅ 图形化界面，操作直观
- ✅ 逐步引导，不易出错
- ✅ 实时反馈和进度显示
- ✅ 支持文件选择器

### ⌨️ 命令行模式（CLI）- 适合脚本和自动化

使用命令行参数直接执行操作：

```bash
bizyair upload -n mymodel -t LoRA -p /path/to/model.safetensors -b "Flux.1 D" -cover /path/to/cover.jpg
```

CLI 模式特点：

- ✅ 快速执行，适合脚本
- ✅ 支持 YAML 批量上传
- ✅ 可集成到 CI/CD 流程
- ✅ 支持环境变量配置

---

## 升级到最新版本

BizyAir CLI 支持一键升级到最新版本：

```bash
# 检查更新
bizyair upgrade --check

# 执行升级
sudo bizyair upgrade
```

---

## 快速开始

### 方式一：使用交互式界面（推荐）

1. **启动 TUI**

```bash
bizyair
```

2. **首次使用需要登录**

   - 输入您的 API Key（从 [BizyAir](https://bizyair.ai) 获取）
   - 按 Enter 确认

3. **选择功能**

   - 上传模型：交互式收集参数并上传
   - 我的模型：在浏览器中查看已上传的模型
   - 退出登录：清除本地 API Key
   - 退出程序

4. **上传模型**
   - 按照界面提示逐步输入信息
   - 支持文件选择器或手动输入路径
   - 支持多版本上传
   - 自动显示上传进度和速率

### 方式二：使用命令行

#### 1. 登录

BizyAir CLI 使用 API Key 进行身份验证。要登录您的设备，运行以下命令：

```bash
# 如果您已设置环境变量 SF_API_KEY
bizyair login

# 或使用 --key/-k 选项指定
bizyair login -k $SF_API_KEY
```

#### 2. 上传模型

**单版本上传示例：**

```bash
bizyair upload -n mymodel -t LoRA \
  -p /local/path/model.safetensors \
  -b "Flux.1 D" \
  -cover "/path/to/cover.jpg" \
  --intro "这是一个动漫风格的 LoRA 模型"
```

**多版本上传示例：**

```bash
bizyair upload -n mymodel -t Checkpoint \
  -v "v1.0" -p /path/file1.safetensors -b SDXL --intro "第一版" -cover "/path/cover1.jpg" \
  -v "v2.0" -p /path/file2.safetensors -b "SD 1.5" --intro "第二版" -cover "/path/cover2.jpg"
```

**核心参数说明：**

- `-n, --name`: 模型名称（必填）
- `-t, --type`: 模型类型（必填，如 LoRA、Checkpoint、Controlnet 等）
- `-p, --path`: 模型文件路径（必填，可多次指定）
- `-b, --base`: 基础模型（必填，如 "Flux.1 D"、SDXL、"SD 1.5" 等）
- `-cover`: 封面文件或 URL（必填）
- `-i, --intro`: 模型介绍文本（必填）
- `--intro-path`: 从文件导入介绍（与 `-i` 二选一）
- `-v, --version`: 版本名称（可选，默认 v1.0）
- `--public`: 是否公开版本（可选，默认 false）

#### 3. 封面上传（必填）

封面支持**本地文件**和 **URL** 两种方式，会自动上传到 OSS 并转换为 WebP 格式：

```bash
# 使用本地文件
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  -cover "/path/to/cover.jpg" --intro "介绍文本"

# 使用 URL
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  -cover "https://example.com/cover.jpg" --intro "介绍文本"
```

**支持的格式：**

- 图片：`.jpg`、`.jpeg`、`.png`、`.gif`、`.webp`
- 视频：`.mp4`、`.webm`、`.mov`（最大 100MB）

**WebP 转换：** 图片封面会自动转换为 WebP 格式以优化加载速度。如果转换失败，会自动回退到原始格式。

#### 4. 介绍文本输入

支持两种方式输入模型介绍（最多 5000 字）：

```bash
# 方式1：直接输入文本
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  -cover cover.jpg --intro "这是模型的详细介绍..."

# 方式2：从文件导入（支持 .txt 和 .md）
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  -cover cover.jpg --intro-path intro.md
```

#### 5. YAML 批量上传

使用 YAML 配置文件可以一次上传多个模型：

```bash
bizyair upload -f config.yaml
```

**YAML 配置示例：**

```yaml
models:
  - name: "anime_style_lora"
    type: "LoRA"
    versions:
      - name: "v1.0"
        base_model: "Flux.1 D"
        model_path: "models/anime_v1.safetensors"
        cover_path: "covers/anime_v1.jpg"
        intro: "第一版动漫风格模型"
        public: true

      - name: "v2.0"
        base_model: "Flux.1 D"
        model_path: "models/anime_v2.safetensors"
        cover_url: "https://example.com/cover.jpg"
        intro_path: "descriptions/v2_intro.txt"
        public: false

  - name: "realistic_checkpoint"
    type: "Checkpoint"
    versions:
      - base_model: "SD 1.5"
        model_path: "checkpoints/realistic.ckpt"
        cover_path: "covers/realistic.webp"
        intro_path: "descriptions/realistic.md"
```

**YAML 配置说明：**

- `model_path`: 模型文件路径（相对于 YAML 文件）
- `cover_path` / `cover_url`: 封面文件或 URL（二选一）
- `intro` / `intro_path`: 介绍文本或文件（二选一）
- `name`: 版本名称（可选，自动递增）
- `public`: 是否公开（可选，默认 false）

详细配置说明请参考 [example.yaml](./example.yaml)

#### 6. 断点续传

上传中断后，重新运行相同命令会自动从断点继续：

```bash
# 第一次上传（中断）
bizyair upload -n mymodel -t Checkpoint -p large-model.safetensors -b SDXL \
  -cover cover.jpg --intro "介绍"
# ^C (中断)

# 重新运行相同命令，自动续传
bizyair upload -n mymodel -t Checkpoint -p large-model.safetensors -b SDXL \
  -cover cover.jpg --intro "介绍"
# ✓ 从断点继续上传
```

Checkpoint 文件保存在 `~/.bizyair/uploads/` 目录。

#### 7. 查看和管理模型

```bash
# 查看我的模型（在浏览器中打开）
bizyair model ls

# 删除模型
bizyair model rm -n mymodel -t Checkpoint
```

#### 8. 退出登录

```bash
bizyair logout
```

---

## 支持的模型类型和基础模型

### 模型类型

- `Checkpoint` - 完整模型检查点
- `LoRA` - 低秩适应模型
- `Controlnet` - 控制网络
- `VAE` - 变分自编码器
- `UNet` - U-Net 模型
- `CLIP` - CLIP 模型
- `Upscaler` - 超分辨率模型
- `Detection` - 检测模型
- `Other` - 其他类型

### 基础模型

- `Flux.1 D` - Flux.1 D 模型
- `Flux.1 Kontext` - Flux.1 Kontext 模型
- `SDXL` - Stable Diffusion XL
- `SD 1.5` - Stable Diffusion 1.5
- `SD 3.5` - Stable Diffusion 3.5
- `Pony` - Pony Diffusion
- `Kolors` - Kolors 模型
- `Hunyuan 1` - 混元 1
- `WAN Video` - WAN Video 模型
- `Qwen-Image` - Qwen 图像模型
- `Other` - 其他基础模型

---

## 详细文档

- [example.yaml](./example.yaml) - YAML 配置示例
- [example.en.yaml](./example.en.yaml) - 英文注释的 YAML 配置示例
- [docs/i18n.md](./docs/i18n.md) - 双语消息目录维护规范

---

## 常见问题

### 如何获取 API Key？

访问 [BizyAir](https://bizyair.ai) 注册并在个人设置中获取 API Key。

### 上传失败怎么办？

1. 检查网络连接
2. 确认 API Key 是否有效
3. 尝试重新运行命令（支持断点续传）

### 如何清除 Checkpoint 文件？

Checkpoint 文件保存在 `~/.bizyair/uploads/` 目录，可以手动删除。

### 封面转换失败怎么办？

如果 WebP 转换失败，系统会自动回退到原始格式，不影响上传。

---

## 贡献

欢迎提交 Issue 和 Pull Request！
