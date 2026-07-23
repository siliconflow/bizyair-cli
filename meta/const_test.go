package meta

import "testing"

func TestManifestURLUsesFixedUpgradeDomain(t *testing.T) {
	const want = "https://storage.bizyair.ai/cli/releases/manifest.json"
	if ManifestURL != want {
		t.Fatalf("ManifestURL = %q, want fixed upgrade URL %q", ManifestURL, want)
	}

	derivedFromBase := StorageDomain + "/cli/releases/manifest.json"
	if DefaultServiceHost != "bizyair.ai" && ManifestURL == derivedFromBase {
		t.Fatalf("ManifestURL must not be derived from DefaultServiceHost %q", DefaultServiceHost)
	}
}
