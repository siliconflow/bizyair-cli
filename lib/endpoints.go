package lib

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

// ServiceEndpoints contains the four BizyAir service origins derived from one
// configurable root domain. Keeping the split here makes a future .vip -> .ai
// migration a one-value change.
type ServiceEndpoints struct {
	Base    string
	API     string
	Meta    string
	Web     string
	Storage string
}

// ResolveServiceEndpoints derives BizyAir service origins from baseDomain.
// Production-like DNS names use api/meta/storage subdomains. A localhost, IP,
// explicit-port, or path-based URL is treated as a unified gateway so local
// and private integration servers can expose every route on one origin.
func ResolveServiceEndpoints(baseDomain string) (ServiceEndpoints, error) {
	baseDomain = strings.TrimSpace(baseDomain)
	if baseDomain == "" {
		baseDomain = meta.DefaultBaseDomain
	}

	u, err := url.Parse(baseDomain)
	if err != nil {
		return ServiceEndpoints{}, i18n.NewError("error.config.base_domain_invalid", map[string]any{"URL": baseDomain}, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ServiceEndpoints{}, i18n.NewError("error.config.base_domain_absolute", map[string]any{"URL": baseDomain}, nil)
	}

	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	normalized := strings.TrimRight(u.String(), "/")
	hostname := strings.ToLower(u.Hostname())

	if hostname == "localhost" || net.ParseIP(hostname) != nil || u.Port() != "" || u.Path != "" {
		return ServiceEndpoints{
			Base: normalized, API: normalized, Meta: normalized, Web: normalized, Storage: normalized,
		}, nil
	}

	rootHost := hostname
	for _, prefix := range []string{"api.", "meta.", "storage.", "www."} {
		if strings.HasPrefix(rootHost, prefix) {
			rootHost = strings.TrimPrefix(rootHost, prefix)
			break
		}
	}
	if rootHost == "" {
		return ServiceEndpoints{}, i18n.NewError("error.config.base_domain_absolute", map[string]any{"URL": baseDomain}, nil)
	}

	origin := func(host string) string {
		copyURL := *u
		copyURL.Host = host
		copyURL.Path = ""
		return strings.TrimRight(copyURL.String(), "/")
	}

	return ServiceEndpoints{
		Base:    origin(rootHost),
		API:     origin("api." + rootHost),
		Meta:    origin("meta." + rootHost),
		Web:     origin("www." + rootHost),
		Storage: origin("storage." + rootHost),
	}, nil
}

func defaultServiceEndpoints() ServiceEndpoints {
	endpoints, err := ResolveServiceEndpoints(meta.DefaultBaseDomain)
	if err != nil {
		panic(err)
	}
	return endpoints
}

func joinEndpoint(origin, path string) string {
	return strings.TrimRight(origin, "/") + "/" + strings.TrimLeft(path, "/")
}

func (e ServiceEndpoints) BaseModelTypesURL() string {
	return joinEndpoint(e.Meta, "/v1/dict")
}

func (e ServiceEndpoints) MyModelsURL() string {
	return joinEndpoint(e.Web, "/community") + "?path=my"
}

func (e ServiceEndpoints) ModelDetailURL(modelID int64) string {
	return joinEndpoint(e.Web, fmt.Sprintf("/community/models/my/%d", modelID))
}
