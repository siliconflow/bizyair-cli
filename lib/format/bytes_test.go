package format

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"1 byte", 1, "1 B"},
		{"100 bytes", 100, "100 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"1 KiB", 1024, "1.0 KB"},
		{"1.5 KiB", 1536, "1.5 KB"},
		{"1 MiB", 1048576, "1.0 MB"},
		{"1.5 MiB", 1572864, "1.5 MB"},
		{"1 GiB", 1073741824, "1.0 GB"},
		{"1 TiB", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
