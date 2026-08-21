package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siliconflow/bizyair-cli/meta"
)

func TestClientCheckModelExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantExists bool
		wantErr    bool
	}{
		{name: "exists", statusCode: http.StatusOK, wantExists: true},
		{name: "does not exist", statusCode: http.StatusNotFound},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"code":50000,"message":"internal error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			exists, err := NewClient(server.URL, "test-api-key").CheckModelExists("mymodel", "LoRA")
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckModelExists() error = %v, wantErr %v", err, tt.wantErr)
			}
			if exists != tt.wantExists {
				t.Fatalf("CheckModelExists() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestGetBaseModelTypes(t *testing.T) {
	t.Run("dict endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":20000,"data":{"base_models":[{"label":"0010","value":"FLUX.1 D"}]}}`))
		}))
		defer server.Close()

		resp, err := NewClient(server.URL, "test-api-key").GetBaseModelTypes()
		if err != nil {
			t.Fatalf("GetBaseModelTypes() error = %v", err)
		}
		if len(resp.Data) != 1 || resp.Data[0].Value != "FLUX.1 D" {
			t.Fatalf("GetBaseModelTypes() = %#v", resp.Data)
		}
	})

	t.Run("falls back to local list on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Forbidden"))
		}))
		defer server.Close()

		resp, err := NewClient(server.URL, "test-api-key").GetBaseModelTypes()
		if err != nil {
			t.Fatalf("GetBaseModelTypes() error = %v", err)
		}
		if len(resp.Data) != len(meta.SupportedBaseModels) {
			t.Fatalf("GetBaseModelTypes() returned %d items, want %d", len(resp.Data), len(meta.SupportedBaseModels))
		}
	})
}

func TestHandleResponse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		body, err := json.Marshal(Response[UserInfo]{
			Code: meta.OKCode,
			Data: UserInfo{Id: "123"},
		})
		if err != nil {
			t.Fatal(err)
		}

		result, err := handleResponse[UserInfo](body)
		if err != nil {
			t.Fatalf("handleResponse() error = %v", err)
		}
		if result.Data.Id != "123" {
			t.Fatalf("handleResponse().Data.Id = %q, want %q", result.Data.Id, "123")
		}
	})

	t.Run("server error code", func(t *testing.T) {
		body, err := json.Marshal(Response[UserInfo]{
			Code:    20004,
			Message: "invalid api key",
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := handleResponse[UserInfo](body); err == nil {
			t.Fatal("handleResponse() returned nil for a server error code")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := handleResponse[UserInfo]([]byte("invalid json")); err == nil {
			t.Fatal("handleResponse() returned nil for invalid JSON")
		}
	})
}
