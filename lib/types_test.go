package lib

import (
	"encoding/json"
	"testing"
)

func TestBizyModelDetailVersionReadsDescription(t *testing.T) {
	var detail BizyModelDetail
	if err := json.Unmarshal([]byte(`{"versions":[{"description":"visible introduction"}]}`), &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Versions) != 1 || detail.Versions[0].Description != "visible introduction" {
		t.Fatalf("detail versions = %#v", detail.Versions)
	}
}
