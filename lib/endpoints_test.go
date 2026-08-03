package lib

import "testing"

func TestResolveServiceEndpoints(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ServiceEndpoints
		wantErr bool
	}{
		{
			name:  "production root",
			input: "https://bizyair.vip",
			want: ServiceEndpoints{
				Base: "https://bizyair.vip", API: "https://api.bizyair.vip", Meta: "https://meta.bizyair.vip",
				Web: "https://www.bizyair.vip", Storage: "https://storage.bizyair.vip",
			},
		},
		{
			name:  "api origin normalizes to root",
			input: "https://api.bizyair.vip/",
			want: ServiceEndpoints{
				Base: "https://bizyair.vip", API: "https://api.bizyair.vip", Meta: "https://meta.bizyair.vip",
				Web: "https://www.bizyair.vip", Storage: "https://storage.bizyair.vip",
			},
		},
		{
			name:  "future root suffix uses the same derivation",
			input: "https://bizyair.ai",
			want: ServiceEndpoints{
				Base: "https://bizyair.ai", API: "https://api.bizyair.ai", Meta: "https://meta.bizyair.ai",
				Web: "https://www.bizyair.ai", Storage: "https://storage.bizyair.ai",
			},
		},
		{
			name:  "web origin normalizes without duplicating prefix",
			input: "https://www.bizyair.ai/",
			want: ServiceEndpoints{
				Base: "https://bizyair.ai", API: "https://api.bizyair.ai", Meta: "https://meta.bizyair.ai",
				Web: "https://www.bizyair.ai", Storage: "https://storage.bizyair.ai",
			},
		},
		{
			name:  "local integration gateway",
			input: "http://127.0.0.1:8080",
			want: ServiceEndpoints{
				Base: "http://127.0.0.1:8080", API: "http://127.0.0.1:8080", Meta: "http://127.0.0.1:8080",
				Web: "http://127.0.0.1:8080", Storage: "http://127.0.0.1:8080",
			},
		},
		{name: "relative URL", input: "bizyair.vip", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveServiceEndpoints(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveServiceEndpoints() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("ResolveServiceEndpoints() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestServiceEndpointURLs(t *testing.T) {
	endpoints, err := ResolveServiceEndpoints("https://bizyair.vip")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"base models":  endpoints.BaseModelTypesURL(),
		"my models":    endpoints.MyModelsURL(),
		"model detail": endpoints.ModelDetailURL(42),
	}
	wants := map[string]string{
		"base models":  "https://www.bizyair.vip/api/special/community/base_model_types",
		"my models":    "https://www.bizyair.vip/community?path=my",
		"model detail": "https://www.bizyair.vip/community/models/my/42",
	}
	for name, got := range tests {
		if got != wants[name] {
			t.Errorf("%s URL = %q, want %q", name, got, wants[name])
		}
	}
}
