package filehash

import (
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
	tabECMA := crc64.MakeTable(crc64.ECMA)
	hashCRC := crc64.New(tabECMA)

	file, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	if _, err := io.Copy(hashCRC, file); err != nil {
		return "", "", err
	}
	crc1 := hashCRC.Sum64()

	if _, err := file.Seek(0, 0); err != nil {
		return "", "", err
	}

	hashMD5 := md5.New()
	if _, err := io.Copy(hashMD5, file); err != nil {
		return "", "", err
	}
	md5Str := base64.StdEncoding.EncodeToString(hashMD5.Sum(nil))

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s%d", md5Str, crc1)))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, md5Str, nil
}
