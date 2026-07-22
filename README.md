<h1 align="center">BizyAir CLI</h1>

<p align="center"><a href="./README.zh-CN.md">简体中文</a> | <strong>English</strong></p>

BizyAir CLI is an open-source command-line tool. Download the latest release from [GitHub](https://github.com/siliconflow/bizyair-cli).

## Overview

BizyAir CLI manages model files on BizyAir. It provides both an interactive terminal interface (TUI) and regular command-line commands (CLI), making it suitable for guided use, scripts, and automation.

### Key features

- Interactive TUI with a guided upload workflow and file picker
- Resumable uploads using local checkpoints
- Up to three concurrent uploads
- Automatic multipart upload for files larger than 100 MB
- Image and video cover support with automatic WebP conversion
- Direct or `.txt`/`.md` model introductions
- YAML batch uploads for multiple models and versions
- VPN detection and upload-performance warnings
- Live progress, transfer rate, and per-version status
- Built-in update checks, installation, and safe rollback
- Standard English and Simplified Chinese interfaces

## Releases

See all published versions on the [GitHub Releases page](https://github.com/siliconflow/bizyair-cli/releases).

## Installation

BizyAir CLI supports Linux, macOS, and Windows. Prebuilt binaries for each platform are available on the [GitHub Releases page](https://github.com/siliconflow/bizyair-cli/releases).

### Linux

1. Download the latest release:

   ```bash
   # Replace this with the latest version
   VERSION=0.2.1

   # Linux amd64
   wget https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-linux-amd64.tar.gz

   # Linux arm64
   # wget https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-linux-arm64.tar.gz
   ```

2. Extract and install:

   ```bash
   tar -xzvf bizyair-v${VERSION}-linux-amd64.tar.gz
   sudo install bizyair /usr/local/bin/bizyair

   # User-only installation without sudo:
   # mkdir -p ~/.local/bin
   # install bizyair ~/.local/bin/bizyair
   # Ensure ~/.local/bin is on PATH.
   ```

3. Verify the installation:

   ```bash
   bizyair --version
   ```

### macOS

1. Download the latest release:

   ```bash
   VERSION=0.2.1

   # Apple Silicon (M1/M2/M3 and newer)
   curl -LO https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-macos-arm64.tar.gz

   # Intel macOS
   # curl -LO https://github.com/siliconflow/bizyair-cli/releases/download/v${VERSION}/bizyair-v${VERSION}-macos-amd64.tar.gz
   ```

2. Extract and install:

   ```bash
   tar -xzvf bizyair-v${VERSION}-macos-arm64.tar.gz
   chmod +x bizyair
   sudo install bizyair /usr/local/bin/bizyair

   # User-only installation without sudo:
   # mkdir -p ~/.local/bin
   # install bizyair ~/.local/bin/bizyair
   ```

3. If macOS displays a security warning on first launch, remove the quarantine attribute:

   ```bash
   sudo xattr -d com.apple.quarantine /usr/local/bin/bizyair
   ```

   Alternatively, open **System Settings → Privacy & Security**, find the blocked `bizyair` executable, and choose **Open Anyway**.

4. Verify the installation:

   ```bash
   bizyair --version
   ```

### Windows

1. Download `bizyair-v{VERSION}-windows-amd64.zip` from the Releases page.
2. Extract `bizyair.exe` to a directory such as `C:\Program Files\BizyAir\`.
3. Add that directory to the system `Path` environment variable:
   - Open **This PC → Properties → Advanced system settings**.
   - Select **Environment Variables**.
   - Edit the system `Path`, add the BizyAir directory, and save.
4. Open Command Prompt or PowerShell and verify the installation:

   ```powershell
   bizyair --version
   ```

   Without a `Path` entry, use the full executable path:

   ```powershell
   C:\Program Files\BizyAir\bizyair.exe --version
   ```

## Building from source

BizyAir CLI requires Go 1.24 or newer.

1. Clone this repository.
2. Build and run the executable:

   ```shell
   make build && ./execs/bizyair
   ```

Platform targets:

- Windows: `make build_windows`
- macOS: `make build_mac`
- Linux amd64: `make build_linux`
- Linux arm64: `make build_linux_arm64`

## Interface language

BizyAir CLI supports English (`en`) and Simplified Chinese (`zh-CN`). It resolves the language once at startup; it does not persist a preference and does not switch languages inside the TUI.

The precedence is:

```text
--lang > system language > en
```

Without `--lang`, Simplified and Traditional Chinese system locales both use the Simplified Chinese interface. All other, empty, or unrecognized system languages use English.

```bash
# Global options must appear before the subcommand
bizyair --lang en upload --help
bizyair --lang zh-CN upgrade --check
```

`--lang` accepts only `en` and `zh-CN`. System locales such as `zh-CN`, `zh-Hans`, `zh-TW`, and `zh-Hant` are detected automatically and all select the Simplified Chinese interface. An unsupported explicit value returns an error listing the supported values.

## Usage modes

### Interactive TUI

Run BizyAir without a subcommand:

```bash
bizyair
```

The TUI provides guided input, immediate validation, file selection, VPN warnings, multi-version uploads, and live progress. It is the recommended starting point for new users.

### Command-line mode

Pass a subcommand and options for scripting or automation:

```bash
bizyair upload -n mymodel -t LoRA -p /path/to/model.safetensors \
  -b "Flux.1 D" --cover /path/to/cover.jpg --intro "Model introduction"
```

CLI mode supports YAML batches, environment variables, and CI/CD integration.

## Updating BizyAir CLI

```bash
# Check without installing
bizyair upgrade --check

# Install the latest release
sudo bizyair upgrade
```

## Quick start

### Option 1: interactive TUI

1. Run `bizyair`.
2. On first use, enter an API key obtained from [BizyAir](https://bizyair.cn) and press Enter.
3. Choose an action:
   - Upload model
   - My models
   - Log out
   - Exit
4. Follow the upload steps. You can select files or type paths, add multiple versions, and monitor progress and transfer speed.

### Option 2: command line

#### 1. Log in

BizyAir CLI authenticates with an API key:

```bash
# Uses SF_API_KEY when it is set
bizyair login

# Or provide a key explicitly
bizyair login -k "$SF_API_KEY"
```

#### 2. Upload a model

Single version:

```bash
bizyair upload -n mymodel -t LoRA \
  -p /local/path/model.safetensors \
  -b "Flux.1 D" \
  --cover "/path/to/cover.jpg" \
  --intro "An anime-style LoRA model"
```

Multiple versions:

```bash
bizyair upload -n mymodel -t Checkpoint \
  -v "v1.0" -p /path/file1.safetensors -b SDXL --intro "First release" --cover "/path/cover1.jpg" \
  -v "v2.0" -p /path/file2.safetensors -b "SD 1.5" --intro "Second release" --cover "/path/cover2.jpg"
```

Core options:

- `-n, --name`: model name (required)
- `-t, --type`: model type, such as `LoRA`, `Checkpoint`, or `Controlnet` (required)
- `-p, --path`: model file; repeat for multiple versions (required)
- `-b, --base`: Base Model, such as `Flux.1 D`, `SDXL`, or `SD 1.5` (required)
- `--cover`: local cover file or URL (required)
- `-i, --intro`: introduction text (required unless `--intro-path` is used)
- `--intro-path`: `.txt` or `.md` introduction file
- `-v, --version`: version name (optional; defaults to `v1.0`)
- `--public`: whether the corresponding version is public (optional; defaults to `false`)

#### 3. Cover uploads

Covers are required and may be local files or URLs. Images are converted to WebP before upload when possible.

```bash
# Local cover
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  --cover "/path/to/cover.jpg" --intro "Introduction"

# Remote cover
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  --cover "https://example.com/cover.jpg" --intro "Introduction"
```

Supported formats:

- Images: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`
- Videos: `.mp4`, `.webm`, `.mov` (up to 100 MB)

If WebP conversion fails, the original cover format is used automatically.

#### 4. Model introductions

Introductions may contain up to 5,000 characters:

```bash
# Direct text
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  --cover cover.jpg --intro "A detailed model introduction..."

# Import from .txt or .md
bizyair upload -n mymodel -t LoRA -p model.safetensors -b "Flux.1 D" \
  --cover cover.jpg --intro-path intro.md
```

#### 5. YAML batch uploads

Upload multiple models from one configuration file:

```bash
bizyair upload -f config.yaml
```

```yaml
models:
  - name: "anime_style_lora"
    type: "LoRA"
    versions:
      - name: "v1.0"
        base_model: "Flux.1 D"
        model_path: "models/anime_v1.safetensors"
        cover_path: "covers/anime_v1.jpg"
        intro: "First anime-style model release"
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

Configuration fields:

- `model_path`: model file path, resolved relative to the YAML file
- `cover_path` / `cover_url`: choose one local cover or remote cover
- `intro` / `intro_path`: choose direct text or an introduction file
- `name`: optional version name; generated automatically when omitted
- `public`: optional visibility; defaults to `false`

See [example.en.yaml](./example.en.yaml) for a complete English-commented example. The equivalent Chinese-commented example is [example.yaml](./example.yaml).

#### 6. Resumable uploads

After an interrupted upload, rerun the same command to resume:

```bash
bizyair upload -n mymodel -t Checkpoint -p large-model.safetensors -b SDXL \
  --cover cover.jpg --intro "Introduction"
# Press Ctrl+C

# Run the same command again; uploaded parts are reused.
bizyair upload -n mymodel -t Checkpoint -p large-model.safetensors -b SDXL \
  --cover cover.jpg --intro "Introduction"
```

Checkpoint files are stored under `~/.bizyair/uploads/`.

#### 7. View and manage models

```bash
# Open My Models in a browser
bizyair model ls

# Delete a model
bizyair model rm -n mymodel -t Checkpoint
```

#### 8. Log out

```bash
bizyair logout
```

## Supported model types and Base Models

### Model types

- `Checkpoint` — complete model checkpoint
- `LoRA` — low-rank adaptation model
- `Controlnet` — control network
- `VAE` — variational autoencoder
- `UNet` — U-Net model
- `CLIP` — CLIP model
- `Upscaler` — super-resolution model
- `Detection` — object detection model
- `Other` — another model type

### Common Base Models

- `Flux.1 D`
- `Flux.1 Kontext`
- `SDXL`
- `SD 1.5`
- `SD 3.5`
- `Pony`
- `Kolors`
- `Hunyuan 1`
- `WAN Video`
- `Qwen-Image`
- `Other`

The CLI retrieves the authoritative Base Model list from the API and uses its built-in list only as a fallback.

## Documentation

- [example.en.yaml](./example.en.yaml) — English-commented YAML example
- [example.yaml](./example.yaml) — Chinese-commented YAML example
- [docs/i18n.md](./docs/i18n.md) — translation maintenance rules

## Troubleshooting

### How do I obtain an API key?

Create an account at [BizyAir](https://bizyair.cn) and obtain a key from your account settings.

### What should I do when an upload fails?

1. Check the network connection.
2. Confirm the API key is valid.
3. Check whether a VPN is affecting the connection.
4. Rerun the same command; interrupted file uploads can resume.

### How do I remove checkpoint files?

Checkpoint files are stored in `~/.bizyair/uploads/` and may be deleted manually when no interrupted uploads need to resume.

### What happens if cover conversion fails?

The uploader automatically falls back to the original file format, so the upload can continue.

## Contributing

Issues and pull requests are welcome. When adding user-facing text, update both embedded catalogs and follow [the i18n maintenance guide](./docs/i18n.md).
