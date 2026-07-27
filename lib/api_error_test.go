package lib

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

func TestKnownAPIErrorIsLocalizedAndStructured(t *testing.T) {
	err := handleError([]byte(`{"code":20004,"message":"raw invalid key"}`), http.StatusUnauthorized)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != http.StatusUnauthorized || apiErr.Code != 20004 || apiErr.RawMessage != "raw invalid key" {
		t.Fatalf("structured details were not preserved: %#v", apiErr)
	}
	if debug := FormatError(err, true); !strings.Contains(debug, "status=401") || !strings.Contains(debug, `message="raw invalid key"`) {
		t.Fatalf("verbose diagnostics do not contain the raw response details: %q", debug)
	}

	if configureErr := i18n.Configure(i18n.English); configureErr != nil {
		t.Fatal(configureErr)
	}
	if got := err.Error(); !strings.Contains(got, "Invalid API key") || strings.Contains(got, "raw invalid key") {
		t.Fatalf("unexpected English API error: %q", got)
	}
	if configureErr := i18n.Configure(i18n.SimplifiedChinese); configureErr != nil {
		t.Fatal(configureErr)
	}
	if got := err.Error(); !strings.Contains(got, "API Key 无效") || strings.Contains(got, "raw invalid key") {
		t.Fatalf("unexpected Chinese API error: %q", got)
	}
}

func TestUnknownAPIErrorRetainsResponseDetails(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{name: "json", body: []byte(`{"code":59999,"message":"upstream detail"}`), want: "upstream detail"},
		{name: "plain text", body: []byte(`gateway unavailable`), want: "gateway unavailable"},
		{name: "quoted text", body: []byte(`"gateway unavailable"`), want: "gateway unavailable"},
		{name: "empty", body: nil, want: ""},
	}

	if err := i18n.Configure(i18n.SimplifiedChinese); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleError(tt.body, http.StatusBadGateway)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected APIError, got %T", err)
			}
			if apiErr.HTTPStatus != http.StatusBadGateway || apiErr.RawMessage != tt.want {
				t.Fatalf("unexpected details: %#v", apiErr)
			}
			if got := err.Error(); !strings.Contains(got, "HTTP 502") || (tt.want != "" && !strings.Contains(got, tt.want)) {
				t.Fatalf("unexpected localized error: %q", got)
			}
		})
	}
}

func TestKnownJSONCodeWinsOverHTTP404(t *testing.T) {
	err := handleError([]byte(`{"code":20224,"message":"raw model detail"}`), http.StatusNotFound)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.KnownCodeMessageID != "error.server.model_not_found" {
		t.Fatalf("known code was not retained: %#v", apiErr)
	}
}

func TestAllKnownServerCodesResolveInBothLanguages(t *testing.T) {
	for _, lang := range []i18n.Language{i18n.English, i18n.SimplifiedChinese} {
		if err := i18n.Configure(lang); err != nil {
			t.Fatal(err)
		}
		for code, messageID := range meta.ServerErrorMessageIDs {
			if got := (&APIError{Code: code, KnownCodeMessageID: messageID}).Error(); strings.HasPrefix(got, "[missing translation:") {
				t.Errorf("server code %d has no %s translation for %s", code, lang, messageID)
			}
		}
	}
}
