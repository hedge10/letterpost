package redirect

import "testing"

func TestTarget(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		path   string
		want   string
		wantOK bool
	}{
		{name: "nothing configured, no path", want: "", wantOK: true},
		{name: "no path gives base", base: "https://example.com/thanks", want: "https://example.com/thanks", wantOK: true},
		{name: "path is put on base origin", base: "https://example.com/thanks", path: "/de/danke", want: "https://example.com/de/danke", wantOK: true},
		{name: "base port is kept", base: "http://localhost:8080/thanks", path: "/other", want: "http://localhost:8080/other", wantOK: true},
		{name: "query and fragment are kept", base: "https://example.com/", path: "/thanks?form=contact#top", want: "https://example.com/thanks?form=contact#top", wantOK: true},
		{name: "path without leading slash", base: "https://example.com/", path: "de/danke", want: "https://example.com/de/danke", wantOK: true},
		{name: "protocol-relative url stays on origin", base: "https://example.com/", path: "//evil.com/", want: "https://example.com/evil.com/", wantOK: true},
		{name: "full url stays on origin", base: "https://example.com/", path: "https://evil.com/", want: "https://example.com/https://evil.com/", wantOK: true},
		{name: "backslash stays on origin", base: "https://example.com/", path: `\evil.com`, want: `https://example.com/\evil.com`, wantOK: true},
		{name: "path without base", path: "/thanks", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Target(tt.base, tt.path)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (target %q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Errorf("target = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorTarget(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		path   string
		code   string
		want   string
		wantOK bool
	}{
		{name: "nothing configured, no path", want: "", wantOK: true},
		{name: "base only", base: "https://example.com/thanks", want: "https://example.com/thanks?error=invalid", wantOK: true},
		{name: "path on base origin", base: "https://example.com/thanks", path: "/oops", want: "https://example.com/oops?error=invalid", wantOK: true},
		{name: "query and fragment are kept", base: "https://example.com/", path: "/oops?lang=de#form", want: "https://example.com/oops?error=invalid&lang=de#form", wantOK: true},
		{name: "old error is replaced", base: "https://example.com/thanks?error=x", want: "https://example.com/thanks?error=invalid", wantOK: true},
		{name: "full url stays on origin", base: "https://example.com/", path: "https://evil.com/", want: "https://example.com/https://evil.com/?error=invalid", wantOK: true},
		{name: "unparsable path falls back to base", base: "https://example.com/thanks", path: "/x%zz", want: "https://example.com/thanks?error=invalid", wantOK: true},
		{name: "control character path falls back to base", base: "https://example.com/thanks", path: "/a\x7f", want: "https://example.com/thanks?error=invalid", wantOK: true},
		{name: "path without base", path: "/oops", wantOK: false},
		{name: "code is escaped", base: "https://example.com/", code: "a b&c", want: "https://example.com/?error=a+b%26c", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := tt.code
			if code == "" {
				code = "invalid"
			}

			got, ok := ErrorTarget(tt.base, tt.path, code)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (target %q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Errorf("target = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrigin(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
		wantOK bool
	}{
		{rawURL: "https://example.com/x", want: "https://example.com", wantOK: true},
		{rawURL: "http://localhost:8080/", want: "http://localhost:8080", wantOK: true},
		{rawURL: "ftp://example.com", wantOK: false},
		{rawURL: "/thanks", wantOK: false},
		{rawURL: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.rawURL, func(t *testing.T) {
			got, ok := Origin(tt.rawURL)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (origin %q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Errorf("origin = %q, want %q", got, tt.want)
			}
		})
	}
}
