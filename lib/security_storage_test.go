package lib

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/siliconflow/bizyair-cli/meta"
)

func TestCredentialAndCheckpointFilesUsePrivatePermissions(t *testing.T) {
	if runtime.GOOS == meta.OSWindows {
		t.Skip("POSIX permission bits are not applicable on Windows")
	}

	home := t.TempDir()
	t.Setenv(meta.EnvHome, home)

	folder := NewSfFolder()
	if err := folder.SaveKey("secret-key"); err != nil {
		t.Fatalf("SaveKey() error = %v", err)
	}
	keyPath := filepath.Join(home, meta.SfFolder, meta.SfApiKey)
	assertPermission(t, filepath.Dir(keyPath), 0700)
	assertPermission(t, keyPath, 0600)

	// Existing credentials created by an older version are repaired on read.
	if err := os.Chmod(keyPath, 0644); err != nil {
		t.Fatalf("chmod legacy key: %v", err)
	}
	if _, err := folder.GetKey(); err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	assertPermission(t, keyPath, 0600)

	checkpoint := &CheckpointInfo{
		ObjectKey:       "models/test.safetensors",
		UploadID:        "upload-id",
		FilePath:        "/tmp/test.safetensors",
		FileSize:        42,
		FileSignature:   "signature",
		PartSize:        int64(meta.MultipartPartSize),
		TotalParts:      1,
		CreatedAt:       time.Now(),
		Bucket:          "bucket",
		Region:          "region",
		Endpoint:        "endpoint",
		AccessKeyId:     "access-key-id",
		AccessKeySecret: "access-key-secret",
		SecurityToken:   "security-token",
		Expiration:      time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	}
	if err := SaveCheckpoint(checkpoint); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}
	checkpointPath, err := GetCheckpointFile(checkpoint.FileSignature)
	if err != nil {
		t.Fatalf("GetCheckpointFile() error = %v", err)
	}
	assertPermission(t, filepath.Dir(checkpointPath), 0700)
	assertPermission(t, checkpointPath, 0600)

	// Checkpoints now intentionally contain the original temporary credentials
	// so an interrupted multipart upload can resume with the same authorized
	// object key. Existing permissive files are repaired before credentials are read.
	if err := os.Chmod(checkpointPath, 0644); err != nil {
		t.Fatalf("chmod legacy checkpoint: %v", err)
	}
	loaded, err := LoadCheckpoint(checkpointPath)
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	assertPermission(t, checkpointPath, 0600)
	if loaded.AccessKeyId != checkpoint.AccessKeyId ||
		loaded.AccessKeySecret != checkpoint.AccessKeySecret ||
		loaded.SecurityToken != checkpoint.SecurityToken ||
		loaded.Expiration != checkpoint.Expiration {
		t.Fatal("checkpoint credentials did not round-trip")
	}
}

func assertPermission(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("permissions for %s = %04o, want %04o", path, got, want)
	}
}
