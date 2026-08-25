package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

func hexToANSI(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return ""
	}
	r, errR := strconv.ParseInt(hex[0:2], 16, 0)
	g, errG := strconv.ParseInt(hex[2:4], 16, 0)
	b, errB := strconv.ParseInt(hex[4:6], 16, 0)
	if errR != nil || errG != nil || errB != nil {
		return ""
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}
