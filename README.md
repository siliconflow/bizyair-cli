<h1 align="center">Silicon Cloud CLI</h1>

<p align="center">
<p>

The Silicon Cloud CLI is an open source tool, you can get the latest version from [GitHub](https://github.com/siliconflow/siliconcloud-cli).

## Introduction
The Silicon Cloud CLI is a command line tool for managing your files on silicon cloud. It provides an easy way to upload,  and manage your silicon cloud files.

## CLI Releases

All releases please [click here](https://github.com/siliconflow/siliconcloud--cli/releases).

## Installation
SiliconCloud-CLI is available on Linux, macOS and Windows platforms.
Binaries for Linux, Windows and Mac are available as tarballs in the [release page](https://github.com/siliconcloud-/siliconcloud--cli/releases).

- Linux
  ```shell
    VERSION=0.1.0
    tar -xzvf siliconcloud-cli-linux-$VERSION-amd64.tar.gz
    install siliconcloud /usr/local/bin
  ```

* Via a GO install

  ```shell
  # NOTE: The dev version will be in effect!
  go install github.com/siliconflow/siliconcloud-cli@latest
  ```

## Building From Source

SiliconCloud-CLI is currently using GO v1.22.X or above.
In order to build it from source you must:

1. Clone the repo
2. Build and run the executable

     ```shell
     make build && ./execs/siliconcloud
     ```

---

## Quick start

### Login
The Silicon Cloud CLI uses api-keys to authenticate client. To login your machine, run the following CLI:

```bash
# if you have an environment variable SF_API_KEY set with your api key
siliconcloud login
# or using an option --key,-k
siliconcloud login -k $SF_API_KEY
```

### Logout
To logout your machine, run the following CLI:

```bash
siliconcloud logout
```

### Upload files
To upload files to the silicon cloud, run the following CLI:

```bash
siliconcloud upload -n mymodel -t bizyair/checkpoint -p /local/path/directory-or-file
```

You can specify overwrite flag to overwrite the model if it already exists in the silicon cloud.

```bash
siliconcloud upload -n mymodel -t bizyair/checkpoint -p /local/path/directory-or-file --overwrite
```


You can specify model name, model type and path to upload by using the `-n`, `-t` and `-p` flags respectively.

### View Models
To view all your models in the silicon cloud, run the following CLI:

```bash
siliconcloud model ls -t bizyair/checkpoint
```

You must specify model type by using the `-t` flag.

You can specify public flag `--public` to view all public models in the silicon cloud. By default it will show only your private models.

```bash

### View Model Files
To view all files in a model, run the following CLI:

```bash
siliconcloud model ls-files -n mymodel -t bizyair/checkpoint
```

You can specify public flag `--public` to view all public model files in the silicon cloud. By default, it will show only your private model files.

If you want to see the files in a model in tree view, run the following CLI:
```bash
siliconcloud model ls-files -n mymodel -t bizyair/checkpoint --tree
```

You must specify model name and model type by using the `-n` and `-t` flags respectively.

### Remove Model
To remove model from the silicon cloud, run the following CLI:

```bash
siliconcloud model rm -n mymodel -t checkpoint
```

You must specify model name and model type by using the `-n` and `-t` flags respectively.
