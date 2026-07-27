package filehash

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc64"
	"io"
	"os"
)

// CalculateHash computes a SHA256 signature derived from MD5 + CRC64 of the file,
// and also returns the base64 MD5 string for server-side verification.
func CalculateHash(filePath string) (string, string, error) {
	return CalculateHashContext(context.Background(), filePath)
}

// CalculateHashContext computes both upload hashes in a single file pass while
// allowing callers to interrupt work on large model files.
func CalculateHashContext(ctx context.Context, filePath string) (string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tabECMA := crc64.MakeTable(crc64.ECMA)
	hashCRC := crc64.New(tabECMA)
	hashMD5 := md5.New()

	file, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	buffer := make([]byte, 1024*1024)
	if _, err := copyWithContext(ctx, io.MultiWriter(hashCRC, hashMD5), file, buffer); err != nil {
		return "", "", err
	}
	crc1 := hashCRC.Sum64()

	md5Str := base64.StdEncoding.EncodeToString(hashMD5.Sum(nil))

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s%d", md5Str, crc1)))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, md5Str, nil
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, buffer []byte) (int64, error) {
	reader := &contextReader{ctx: ctx, reader: src}
	return io.CopyBuffer(dst, reader, buffer)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}
