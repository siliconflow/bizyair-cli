package lib

import (
	"testing"
	"time"
)

func TestIsCredentialExpired(t *testing.T) {
	tests := []struct {
		name       string
		expiration string
		want       bool
	}{
		{name: "missing", expiration: "", want: true},
		{name: "invalid", expiration: "not-a-time", want: true},
		{name: "already expired", expiration: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), want: true},
		{name: "inside safety window", expiration: time.Now().Add(4 * time.Minute).UTC().Format(time.RFC3339), want: true},
		{name: "valid", expiration: time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCredentialExpired(tt.expiration); got != tt.want {
				t.Fatalf("IsCredentialExpired(%q) = %v, want %v", tt.expiration, got, tt.want)
			}
		})
	}
}

func TestAliOssStorageClientRetainsCheckpointCredentials(t *testing.T) {
	client, err := NewAliOssStorageClient(
		"oss-us-east-1.aliyuncs.com",
		"bucket",
		"access-key-id",
		"access-key-secret",
		"security-token",
	)
	if err != nil {
		t.Fatalf("NewAliOssStorageClient() error = %v", err)
	}

	expiration := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	client.SetExpiration(expiration)
	if client.ossAccessKeyId != "access-key-id" ||
		client.ossAccessKey != "access-key-secret" ||
		client.ossSecurityToken != "security-token" ||
		client.ossExpiration != expiration {
		t.Fatal("OSS client did not retain checkpoint credentials")
	}
}
