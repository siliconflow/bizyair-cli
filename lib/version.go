package lib

import (
	"fmt"
	"strconv"
	"strings"
)

// Version 语义化版本结构
type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// ParseVersion 解析语义化版本字符串 (v1.2.3 或 1.2.3)
func ParseVersion(ver string) (*Version, error) {
	// 移除 'v' 前缀
	ver = strings.TrimPrefix(ver, "v")
	ver = strings.TrimPrefix(ver, "V")

	parts := strings.Split(ver, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected format: v1.2.3)", ver)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   "v" + ver,
	}, nil
}

// Compare 比较两个版本
// 返回: 1 如果 v > other, -1 如果 v < other, 0 如果相等
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}

	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}

	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}

	return 0
}

// IsNewerThan 判断是否比另一个版本新
func (v *Version) IsNewerThan(other *Version) bool {
	return v.Compare(other) > 0
}

// IsOlderThan 判断是否比另一个版本旧
func (v *Version) IsOlderThan(other *Version) bool {
	return v.Compare(other) < 0
}

// Equals 判断是否与另一个版本相等
func (v *Version) Equals(other *Version) bool {
	return v.Compare(other) == 0
}

// String 返回版本字符串
func (v *Version) String() string {
	return v.Raw
}

// BumpType 版本递增类型
type BumpType string

const (
	BumpMajor BumpType = "major"
	BumpMinor BumpType = "minor"
	BumpPatch BumpType = "patch"
)

// Bump 递增版本号
func (v *Version) Bump(bumpType BumpType) *Version {
	newVersion := &Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}

	switch bumpType {
	case BumpMajor:
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
	case BumpMinor:
		newVersion.Minor++
		newVersion.Patch = 0
	case BumpPatch:
		newVersion.Patch++
	}

	newVersion.Raw = fmt.Sprintf("v%d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)
	return newVersion
}

// CompareVersionStrings 直接比较两个版本字符串
func CompareVersionStrings(v1, v2 string) (int, error) {
	ver1, err := ParseVersion(v1)
	if err != nil {
		return 0, err
	}

	ver2, err := ParseVersion(v2)
	if err != nil {
		return 0, err
	}

	return ver1.Compare(ver2), nil
}
