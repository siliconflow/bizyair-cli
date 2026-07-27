package lib

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/siliconflow/bizyair-cli/meta"
)

type requestSnapshot struct {
	Method        string
	Path          string
	RawQuery      string
	Authorization string
	CLIVersion    string
	ContentType   string
	Body          string
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []requestSnapshot
}

func (r *requestRecorder) record(req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, requestSnapshot{
		Method: req.Method, Path: req.URL.Path, RawQuery: req.URL.RawQuery,
		Authorization: req.Header.Get(meta.HeaderAuthorization),
		CLIVersion:    req.Header.Get(meta.HeaderSiliconCliVersion),
		ContentType:   req.Header.Get(meta.HeaderContentType), Body: string(body),
	})
}

func (r *requestRecorder) snapshot() []requestSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]requestSnapshot(nil), r.requests...)
}

func writeAPIResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": meta.OKCode, "data": data})
}

func TestClientUsesNewServiceEndpointContract(t *testing.T) {
	metaRequests := &requestRecorder{}
	apiRequests := &requestRecorder{}
	webRequests := &requestRecorder{}

	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metaRequests.record(r)
		if strings.HasSuffix(r.URL.Path, "/exists") {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeAPIResponse(w, nil)
	}))
	defer metaServer.Close()
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequests.record(r)
		writeAPIResponse(w, nil)
	}))
	defer apiServer.Close()
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webRequests.record(r)
		writeAPIResponse(w, nil)
	}))
	defer webServer.Close()

	client := NewClientWithEndpoints("integration", ServiceEndpoints{
		API: apiServer.URL, Meta: metaServer.URL, Web: webServer.URL, Storage: webServer.URL,
	}, "test-key")
	ctx := context.Background()

	calls := []func() error{
		func() error { _, err := client.UserInfoContext(ctx); return err },
		func() error { _, err := client.OssSignContext(ctx, "abc123", "LoRA"); return err },
		func() error { _, err := client.CommitFileV2Context(ctx, "sig", "key", "md5", "LoRA"); return err },
		func() error {
			_, err := client.CommitModelV2Context(ctx, "model", "LoRA", []*ModelVersion{{
				Version: "v1.0", Introduction: "visible introduction",
			}})
			return err
		},
		func() error {
			_, err := client.ListModelContext(ctx, 1, 10, "", "Recently", []string{"LoRA"}, nil)
			return err
		},
		func() error { _, err := client.GetBizyModelDetailContext(ctx, 7); return err },
		func() error { _, err := client.DeleteBizyModelByIdContext(ctx, 7); return err },
		func() error { _, err := client.CheckModelExistsContext(ctx, "model", "LoRA"); return err },
		func() error {
			_, err := client.CommitInputResourceContext(ctx, "cover.webp", "inputs/cover.webp")
			return err
		},
		func() error { _, err := client.GetUploadTokenContext(ctx, "cover.webp", "inputs"); return err },
		func() error { _, err := client.GetBaseModelTypesContext(ctx); return err },
	}
	for i, call := range calls {
		if err := call(); err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
	}

	wantMeta := []struct{ method, path, query string }{
		{http.MethodGet, "/v1/user/info", ""},
		{http.MethodGet, "/v1/files/abc123", "type=LoRA"},
		{http.MethodPost, "/v1/files", ""},
		{http.MethodPost, "/v1/bizy_models", ""},
		{http.MethodGet, "/v1/bizy_models/my", "current=1&model_types=LoRA&page_size=10&sort=Recently"},
		{http.MethodGet, "/v1/bizy_models/7/detail", ""},
		{http.MethodDelete, "/v1/bizy_models/7", ""},
		{http.MethodGet, "/v1/bizy_models/exists", "name=model&type=LoRA"},
		{http.MethodPost, "/v1/input_resource/commit", ""},
	}
	gotMeta := metaRequests.snapshot()
	if len(gotMeta) != len(wantMeta) {
		t.Fatalf("meta request count = %d, want %d", len(gotMeta), len(wantMeta))
	}
	for i, want := range wantMeta {
		got := gotMeta[i]
		if got.Method != want.method || got.Path != want.path {
			t.Errorf("meta request %d = %s %s, want %s %s", i, got.Method, got.Path, want.method, want.path)
		}
		if got.RawQuery != want.query {
			t.Errorf("meta request %d query = %q, want %q", i, got.RawQuery, want.query)
		}
		if got.Authorization != "Bearer test-key" {
			t.Errorf("meta request %d authorization = %q", i, got.Authorization)
		}
		if got.CLIVersion != meta.Version {
			t.Errorf("meta request %d CLI version = %q, want %q", i, got.CLIVersion, meta.Version)
		}
	}
	deleteRequest := gotMeta[6]
	if deleteRequest.Body != "" {
		t.Errorf("DELETE body = %q, want empty", deleteRequest.Body)
	}
	var commitModelBody struct {
		Versions []map[string]any `json:"versions"`
	}
	if err := json.Unmarshal([]byte(gotMeta[3].Body), &commitModelBody); err != nil {
		t.Fatalf("decode commit model body: %v", err)
	}
	if len(commitModelBody.Versions) != 1 || commitModelBody.Versions[0]["description"] != "visible introduction" {
		t.Fatalf("commit model versions = %#v, want description field", commitModelBody.Versions)
	}
	if _, exists := commitModelBody.Versions[0]["intro"]; exists {
		t.Fatalf("commit model still sends obsolete intro field: %#v", commitModelBody.Versions[0])
	}
	if _, exists := commitModelBody.Versions[0]["introduction"]; exists {
		t.Fatalf("commit model sends unsupported introduction field: %#v", commitModelBody.Versions[0])
	}

	gotAPI := apiRequests.snapshot()
	if len(gotAPI) != 1 || gotAPI[0].Method != http.MethodGet || gotAPI[0].Path != "/v1/upload/token" {
		t.Fatalf("upload token request = %#v", gotAPI)
	}
	if gotAPI[0].RawQuery != "file_name=cover.webp&file_type=inputs" {
		t.Errorf("upload token query = %q", gotAPI[0].RawQuery)
	}
	if gotAPI[0].Authorization != "Bearer test-key" || gotAPI[0].CLIVersion != meta.Version {
		t.Errorf("upload token headers = %#v", gotAPI[0])
	}
	gotWeb := webRequests.snapshot()
	if len(gotWeb) != 1 || gotWeb[0].Path != "/api/special/community/base_model_types" {
		t.Fatalf("base model request = %#v", gotWeb)
	}
	if gotWeb[0].Authorization != "" || gotWeb[0].CLIVersion != meta.Version {
		t.Errorf("base model headers = %#v", gotWeb[0])
	}
}

func TestClientContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.UserInfoContext(ctx)
		done <- err
	}()
	<-requestStarted
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("UserInfoContext() returned nil after cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop after context cancellation")
	}
}

func TestClientKeepsSafeDefaultTransport(t *testing.T) {
	client := NewClient("https://bizyair.vip", "key")
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.httpClient.Transport)
	}
	if client.httpClient.Timeout <= 0 || transport.Proxy == nil || transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 {
		t.Fatalf("unsafe HTTP client defaults: timeout=%s proxy=%v dial=%v tls_timeout=%s",
			client.httpClient.Timeout, transport.Proxy != nil, transport.DialContext != nil, transport.TLSHandshakeTimeout)
	}
	if !transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 is disabled")
	}
}
