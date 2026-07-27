package lib

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadToTempUsesURLPathExtension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("cover"))
	}))
	defer server.Close()

	path, cleanup, err := DownloadToTempContext(context.Background(), server.URL+"/image.png?token=secret")
	if err != nil {
		t.Fatalf("DownloadToTempContext() error = %v", err)
	}
	defer cleanup()
	if got := filepath.Ext(path); got != ".png" {
		t.Fatalf("temporary extension = %q, want .png", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded cover: %v", err)
	}
	if string(data) != "cover" {
		t.Fatalf("downloaded content = %q", data)
	}
}

func TestDownloadToTempRejectsOversizedContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxRemoteCoverSize+1))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if _, _, err := DownloadToTempContext(context.Background(), server.URL+"/cover.webp"); err == nil {
		t.Fatal("expected oversized cover to be rejected")
	}
}

func TestDownloadToTempHonorsContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, err := DownloadToTempContext(ctx, server.URL+"/cover.webp")
		done <- err
	}()
	<-requestStarted
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("download did not stop after context cancellation")
	}
}
