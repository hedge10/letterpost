// Package redirect builds the target of the redirect after a form
// submission.
package redirect

import (
	"net/url"
	"strings"
)

// Target returns where to redirect after a submission. An empty path gives
// base. Otherwise path is put on the origin of base, so the redirect never
// leaves that origin. ok is false when a path is given but base is not an
// absolute http(s) URL.
func Target(base, path string) (target string, ok bool) {
	if path == "" {
		return base, true
	}

	origin, ok := Origin(base)
	if !ok {
		return "", false
	}

	return origin + "/" + strings.TrimLeft(path, "/"), true
}

// Origin returns scheme://host of rawURL. ok is false unless rawURL is an
// absolute http(s) URL.
func Origin(rawURL string) (origin string, ok bool) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}

	return u.Scheme + "://" + u.Host, true
}
