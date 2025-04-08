package meta

const (
	Name        = "bizyair"
	Description = "A CLI tools for BizyAir."
)

var (
	// Version This variable is replaced in compile time. `-ldflags "-X 'github.com/siliconflow/bizyair-cli/meta.Version=${VERSION}'"`
	Version = "0.0.1"
	// Commit This variable is replaced in compile time. `-ldflags "-X 'github.com/siliconflow/bizyair-cli/meta.Commit=${GIT_REV}'"`
	Commit = "latest"
	// BuildDate This variable is replaced in compile time. `-ldflags "-X 'github.com/siliconflow/bizyair-cli/meta.BuildDate=${BUILD_DATE}'"`
	BuildDate = "2024-07-30T16:47:50+08:00"
)
