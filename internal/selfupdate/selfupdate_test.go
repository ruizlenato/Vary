package selfupdate

import "testing"

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{name: "plain semver", input: "0.1.0", want: "0.1.0", wantOK: true},
		{name: "prefixed semver", input: "v0.1.0", want: "0.1.0", wantOK: true},
		{name: "project tag format", input: "v.0.1.0", want: "0.1.0", wantOK: true},
		{name: "devel", input: "(devel)", want: "", wantOK: false},
		{name: "empty", input: "", want: "", wantOK: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := normalizeVersion(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("version = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReleaseOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
		want string
	}{
		{name: "darwin maps to macos", goos: "darwin", want: "macos"},
		{name: "windows stays windows", goos: "windows", want: "windows"},
		{name: "linux stays linux", goos: "linux", want: "linux"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := releaseOS(tt.goos); got != tt.want {
				t.Fatalf("releaseOS(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}
