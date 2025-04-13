package lib

import (
	"bytes"
	"encoding/base64"
	"image"

	// 显式导入各种image格式的解码器
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/chai2010/webp"
)

func Throttle(fn func(x int), wait time.Duration) func(x int) {
	lastTime := time.Now()
	return func(x int) {
		now := time.Now()
		if now.Sub(lastTime) >= wait {
			fn(x)
			lastTime = now
		}
	}
}

// convert any image type to webp, usr quality 92 by default
func ImageToWebp(inputPath string) (outputFile []byte, newFilename, base64Str string, err error) {
	// 1. read original image
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, "", "", err
	}

	// 2. decode image
	img, _, err := image.Decode(bytes.NewReader(inputData))
	if err != nil {
		return nil, "", "", err
	}

	// 3. encode to webp
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: 92}); err != nil {
		return nil, "", "", err
	}

	// 4. make output
	outputFile = buf.Bytes()

	// 5. encode to base64
	base64Str = base64.StdEncoding.EncodeToString(outputFile)

	// 6. name the output webp file to save

	newFilename = inputPath + ".bizyair-temp.webp"

	// 7. save webp file
	if err := os.WriteFile(newFilename, outputFile, 0644); err != nil {
		return nil, "", "", err
	}

	return outputFile, newFilename, base64Str, nil
}
