<h1 align="center">BizyAir-cli</h1>

<p align="center">
<p>

The BizyAir CLI is an open source tool, you can get the latest version from [GitHub](https://github.com/siliconflow/bizyair-cli).

## Introduction
The BizyAir CLI is a command line tool for managing your files on bizyair. It provides an easy way to upload,  and manage your bizyair files.

## CLI Releases

All releases please [click here](https://github.com/siliconflow/bizyair-cli/releases).

## Installation
BizyAir-CLI is available on Linux, macOS and Windows platforms.
Binaries for Linux, Windows and Mac are available as tarballs in the [release page](https://github.com/siliconflow/bizyair-cli/releases).

- Linux
  ```shell
    VERSION=0.1.0
    tar -xzvf bizyair-cli-linux-$VERSION-amd64.tar.gz
    install bizyair /usr/local/bin
  ```

* Via a GO install

  ```shell
  # NOTE: The dev version will be in effect!
  go install github.com/siliconflow/bizyair-cli@latest
  ```

## Building From Source

BizyAir-CLI is currently using GO v1.22.X or above.
In order to build it from source you must:

1. Clone the repo
2. Build and run the executable

     ```shell
     make build && ./execs/bizyair
     ```

- For Windows OS, run `make build_windows`.
- For MacOS, run `make build_mac`.
- For Linux, run `make build_linux` or `make build_linux_arm64`.
---

## Quick start

### Login
The BizyAir CLI uses api-keys to authenticate client. To login your machine, run the following CLI:

```bash
# if you have an environment variable SF_API_KEY set with your api key
bizyair login
# or using an option --key,-k
bizyair login -k $SF_API_KEY
```

### Logout
To logout your machine, run the following CLI:

```bash
bizyair logout
```

### Upload files
To upload files to the silicon cloud, run the following CLI:

```bash
bizyair upload -n mymodel -t [LoRA | Controlnet | Checkpoint] -p /local/path/file -b [basemodel]
```

~~You can specify overwrite flag to overwrite the model if it already exists in the silicon cloud.~~

```bash
(Deprecated) bizyair upload -n mymodel -t bizyair/checkpoint -p /local/path/file --overwrite
```

You can specify model name, model type, base model and path to upload by using the `-n`, `-t`, `-b` and `-p` flags respectively.

You can specify version names, introductions, and model cover urls using `-v`, `-i`, `-cover` flags respectively.

To upload multiple covers for one version, use ";" as a separator.

To upload multiple versions of your model, you can provide a list of values for each flag. Each value in the list corresponds to a specific version of your model. For example, consider the following usage:

```bash
bizyair upload -n mymodel -t Checkpoint \
-v "v1" -b SDXL -p /local/path/file1 -i "sdxl checkpoint1" -cover "${url1};${url2}" \
-v "v2" -b [Basemodel2] -p /local/path/file2 -cover "${url3}" \
-v "v3" -b [basemodel3] -p /local/path/file3 \
...
```
 
Default version names will be used if `-v` is not specified.


### View Models (Not Supported Yet)
To view all your models in the silicon cloud, run the following CLI:

```bash
bizyair model ls -t Checkpoint
```

You must specify model type by using the `-t` flag.

You can specify public flag `--public` to view all public models in the silicon cloud. By default it will show only your private models.

```bash

### View Model Files
To view all files in a model, run the following CLI:

```bash
bizyair model ls-files -n mymodel -t bizyair/checkpoint
```

You can specify public flag `--public` to view all public model files in the silicon cloud. By default, it will show only your private model files.

If you want to see the files in a model in tree view, run the following CLI:
```bash
bizyair model ls-files -n mymodel -t Checkpoint --tree
```

You must specify model name and model type by using the `-n` and `-t` flags respectively.

### Remove Model
To remove model from the silicon cloud, run the following CLI:

```bash
bizyair model rm -n mymodel -t Checkpoint
```

You must specify model name and model type by using the `-n` and `-t` flags respectively.
