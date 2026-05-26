package lib

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siliconflow/bizyair-cli/meta"
)

// helper to create a test server and client
func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := NewClient(server.URL, "test-api-key")
	return client, server
}

func TestClient_UserInfo_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got %q", r.Header.Get("Authorization"))
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		resp := Response[UserInfo]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: UserInfo{
				Id:    "user-123",
				Name:  "TestUser",
				Email: "test@example.com",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.UserInfo()
	if err != nil {
		t.Fatalf("UserInfo() error = %v", err)
	}
	if result.Data.Id != "user-123" {
		t.Errorf("UserInfo().Data.Id = %q, want %q", result.Data.Id, "user-123")
	}
	if result.Data.Name != "TestUser" {
		t.Errorf("UserInfo().Data.Name = %q, want %q", result.Data.Name, "TestUser")
	}
}

func TestClient_UserInfo_ServerError(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":50000,"message":"internal server error"}`))
	})
	defer server.Close()

	_, err := client.UserInfo()
	if err == nil {
		t.Fatal("UserInfo() should return error on 500")
	}
}

func TestClient_UserInfo_Unauthorized(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"code":20004,"message":"Invalid api key"}`))
	})
	defer server.Close()

	_, err := client.UserInfo()
	if err == nil {
		t.Fatal("UserInfo() should return error on 401")
	}
}

func TestClient_OssSign_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		resp := Response[FilesResp]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: FilesResp{
				File: &FileInfo{
					Sign:      "test-signature",
					ObjectKey: "test-object-key",
					Id:        123,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.OssSign("abc123", "LoRA")
	if err != nil {
		t.Fatalf("OssSign() error = %v", err)
	}
	if result.Data.File.Sign != "test-signature" {
		t.Errorf("OssSign().Data.File.Sign = %q, want %q", result.Data.File.Sign, "test-signature")
	}
}

func TestClient_CommitFileV2_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := Response[FilesResp]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: FilesResp{
				File: &FileInfo{
					Id: 456,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.CommitFileV2("sig", "obj-key", "md5hash", "LoRA")
	if err != nil {
		t.Fatalf("CommitFileV2() error = %v", err)
	}
	if result.Data.File.Id != 456 {
		t.Errorf("CommitFileV2().Data.File.Id = %d, want 456", result.Data.File.Id)
	}
}

func TestClient_CommitModelV2_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := Response[ModelCommitResp]{
			Code:    meta.OKCode,
			Message: "ok",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	versions := []*ModelVersion{
		{Version: "v1.0", BaseModel: "SDXL", Sign: "test-sign"},
	}
	result, err := client.CommitModelV2("mymodel", "LoRA", versions)
	if err != nil {
		t.Fatalf("CommitModelV2() error = %v", err)
	}
	if result.Code != meta.OKCode {
		t.Errorf("CommitModelV2().Code = %d, want %d", result.Code, meta.OKCode)
	}
}

func TestClient_ListModel_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := Response[BizyModelListResp]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: BizyModelListResp{
				Total:   1,
				Current: 1,
				List: []*BizyModelInfo{
					{Id: 1, Name: "test-model", Type: "LoRA"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.ListModel(1, 10, "", "", nil, nil)
	if err != nil {
		t.Fatalf("ListModel() error = %v", err)
	}
	if result.Data.Total != 1 {
		t.Errorf("ListModel().Data.Total = %d, want 1", result.Data.Total)
	}
}

func TestClient_DeleteBizyModelById_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		resp := Response[interface{}]{
			Code:    meta.OKCode,
			Message: "ok",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	_, err := client.DeleteBizyModelById(123)
	if err != nil {
		t.Fatalf("DeleteBizyModelById() error = %v", err)
	}
}

func TestClient_CheckModelExists_True(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	exists, err := client.CheckModelExists("mymodel", "LoRA")
	if err != nil {
		t.Fatalf("CheckModelExists() error = %v", err)
	}
	if !exists {
		t.Error("CheckModelExists() should return true for 200")
	}
}

func TestClient_CheckModelExists_False(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	exists, err := client.CheckModelExists("mymodel", "LoRA")
	if err != nil {
		t.Fatalf("CheckModelExists() error = %v", err)
	}
	if exists {
		t.Error("CheckModelExists() should return false for 404")
	}
}

func TestClient_CheckModelExists_Error(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":50000,"message":"internal error"}`))
	})
	defer server.Close()

	_, err := client.CheckModelExists("mymodel", "LoRA")
	if err == nil {
		t.Fatal("CheckModelExists() should return error for 500")
	}
}

func TestClient_RemoveModel_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		resp := Response[ModelDeleteResp]{
			Code:    meta.OKCode,
			Message: "ok",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	_, err := client.RemoveModel("LoRA", "mymodel")
	if err != nil {
		t.Fatalf("RemoveModel() error = %v", err)
	}
}

func TestClient_GetUploadToken_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		resp := Response[FilesResp]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: FilesResp{
				File: &FileInfo{
					Id:        789,
					ObjectKey: "uploads/test.zip",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.GetUploadToken("test.zip", "cli")
	if err != nil {
		t.Fatalf("GetUploadToken() error = %v", err)
	}
	if result.Data.File.Id != 789 {
		t.Errorf("GetUploadToken().Data.File.Id = %d, want 789", result.Data.File.Id)
	}
}

func TestClient_CommitInputResource_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		resp := Response[InputResourceCommitResp]{
			Code:    meta.OKCode,
			Message: "ok",
			Data: InputResourceCommitResp{
				Id:   100,
				Name: "cover",
				Url:  "https://cdn.example.com/cover.webp",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	result, err := client.CommitInputResource("cover", "uploads/cover.webp")
	if err != nil {
		t.Fatalf("CommitInputResource() error = %v", err)
	}
	if result.Data.Url != "https://cdn.example.com/cover.webp" {
		t.Errorf("CommitInputResource().Data.Url = %q, want %q", result.Data.Url, "https://cdn.example.com/cover.webp")
	}
}

func TestHandleError_NotFound(t *testing.T) {
	err := handleError([]byte(""), http.StatusNotFound)
	if err == nil {
		t.Fatal("handleError should return error for 404")
	}
}

func TestHandleError_KnownServerCode(t *testing.T) {
	body, _ := json.Marshal(Response[interface{}]{Code: 20004, Message: "invalid api key"})
	err := handleError(body, http.StatusUnauthorized)
	if err == nil {
		t.Fatal("handleError should return error for known server code")
	}
}

func TestHandleError_InvalidJSON(t *testing.T) {
	err := handleError([]byte(`"some plain text error"`), http.StatusInternalServerError)
	if err == nil {
		t.Fatal("handleError should return error for non-JSON response")
	}
}

func TestHandleError_EmptyBody(t *testing.T) {
	err := handleError([]byte(""), http.StatusInternalServerError)
	if err == nil {
		t.Fatal("handleError should return error for empty body")
	}
}

func TestHandleResponse_Success(t *testing.T) {
	body, _ := json.Marshal(Response[UserInfo]{
		Code:    meta.OKCode,
		Message: "ok",
		Data:    UserInfo{Id: "123"},
	})

	result, err := handleResponse[UserInfo](body)
	if err != nil {
		t.Fatalf("handleResponse() error = %v", err)
	}
	if result.Data.Id != "123" {
		t.Errorf("handleResponse().Data.Id = %q, want %q", result.Data.Id, "123")
	}
}

func TestHandleResponse_ServerErrorCode(t *testing.T) {
	body, _ := json.Marshal(Response[UserInfo]{
		Code:    20004,
		Message: "invalid api key",
	})

	_, err := handleResponse[UserInfo](body)
	if err == nil {
		t.Fatal("handleResponse should return error for server error code")
	}
}

func TestHandleResponse_InvalidJSON(t *testing.T) {
	_, err := handleResponse[UserInfo]([]byte("invalid json"))
	if err == nil {
		t.Fatal("handleResponse should return error for invalid JSON")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("https://api.example.com", "my-key")
	if client.Domain != "https://api.example.com" {
		t.Errorf("Domain = %q, want %q", client.Domain, "https://api.example.com")
	}
	if client.ApiKey != "my-key" {
		t.Errorf("ApiKey = %q, want %q", client.ApiKey, "my-key")
	}
}
