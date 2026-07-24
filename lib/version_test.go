package lib

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{"standard v-prefixed", "v1.2.3", &Version{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3"}, false},
		{"no v-prefix", "1.2.3", &Version{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3"}, false},
		{"uppercase V-prefix", "V2.0.1", &Version{Major: 2, Minor: 0, Patch: 1, Raw: "v2.0.1"}, false},
		{"zero version", "v0.0.0", &Version{Major: 0, Minor: 0, Patch: 0, Raw: "v0.0.0"}, false},
		{"large numbers", "v99.99.99", &Version{Major: 99, Minor: 99, Patch: 99, Raw: "v99.99.99"}, false},
		{"missing patch", "v1.2", nil, true},
		{"missing minor and patch", "v1", nil, true},
		{"empty string", "", nil, true},
		{"non-numeric major", "va.2.3", nil, true},
		{"non-numeric minor", "v1.b.3", nil, true},
		{"non-numeric patch", "v1.2.c", nil, true},
		{"too many parts", "v1.2.3.4", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("ParseVersion(%q) = {Major:%d, Minor:%d, Patch:%d}, want {Major:%d, Minor:%d, Patch:%d}",
					tt.input, got.Major, got.Minor, got.Patch, tt.want.Major, tt.want.Minor, tt.want.Patch)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("ParseVersion(%q).Raw = %q, want %q", tt.input, got.Raw, tt.want.Raw)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	v1_0_0 := &Version{Major: 1, Minor: 0, Patch: 0}
	v1_0_1 := &Version{Major: 1, Minor: 0, Patch: 1}
	v1_0_99 := &Version{Major: 1, Minor: 0, Patch: 99}
	v1_1_0 := &Version{Major: 1, Minor: 1, Patch: 0}
	v2_0_0 := &Version{Major: 2, Minor: 0, Patch: 0}

	tests := []struct {
		name  string
		v     *Version
		other *Version
		want  int
	}{
		{"equal", v1_0_0, v1_0_0, 0},
		{"patch newer", v1_0_1, v1_0_0, 1},
		{"patch older", v1_0_0, v1_0_1, -1},
		{"minor newer", v1_1_0, v1_0_0, 1},
		{"minor older", v1_0_0, v1_1_0, -1},
		{"major newer", v2_0_0, v1_0_0, 1},
		{"major older", v1_0_0, v2_0_0, -1},
		{"major beats minor", v2_0_0, v1_1_0, 1},
		{"minor beats patch", v1_1_0, v1_0_99, 1}, // can't construct v1.0.99 easily, use literal
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.Compare(tt.other)
			if got != tt.want {
				t.Errorf("Compare() = %d, want %d", got, tt.want)
			}
		})
	}

	if !v2_0_0.IsNewerThan(v1_0_0) {
		t.Error("v2.0.0 should be newer than v1.0.0")
	}
	if v1_0_0.IsNewerThan(v2_0_0) {
		t.Error("v1.0.0 should not be newer than v2.0.0")
	}
}

func TestVersion_Bump(t *testing.T) {
	v := &Version{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3"}

	tests := []struct {
		name     string
		bumpType BumpType
		want     *Version
	}{
		{"bump major", BumpMajor, &Version{Major: 2, Minor: 0, Patch: 0, Raw: "v2.0.0"}},
		{"bump minor", BumpMinor, &Version{Major: 1, Minor: 3, Patch: 0, Raw: "v1.3.0"}},
		{"bump patch", BumpPatch, &Version{Major: 1, Minor: 2, Patch: 4, Raw: "v1.2.4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.Bump(tt.bumpType)
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("Bump(%s) = {Major:%d, Minor:%d, Patch:%d}, want {Major:%d, Minor:%d, Patch:%d}",
					tt.bumpType, got.Major, got.Minor, got.Patch, tt.want.Major, tt.want.Minor, tt.want.Patch)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("Bump(%s).Raw = %q, want %q", tt.bumpType, got.Raw, tt.want.Raw)
			}
		})
	}
}

func TestCompareVersionStrings(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{"v1 newer", "v2.0.0", "v1.0.0", 1, false},
		{"v1 older", "v1.0.0", "v2.0.0", -1, false},
		{"equal", "v1.0.0", "v1.0.0", 0, false},
		{"invalid v1", "invalid", "v1.0.0", 0, true},
		{"invalid v2", "v1.0.0", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersionStrings(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersionStrings(%q, %q) error = %v, wantErr %v", tt.v1, tt.v2, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersionStrings(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
