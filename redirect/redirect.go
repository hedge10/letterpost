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

// ErrorTarget returns where to redirect after a failed submission. It is
// Target(base, path) with error=code set in the query, replacing any error
// already there. When that target cannot be parsed, base itself is used. When
// neither base nor path is configured it returns an empty target, as there is
// nowhere to redirect to. ok is false when Target fails (a path without a
// valid base) or when not even base is usable.
func ErrorTarget(base, path, code string) (target string, ok bool) {
	target, ok = Target(base, path)
	if !ok || target == "" {
		return "", ok
	}

	u, err := url.Parse(target)
	if err != nil {
		target, _ = Target(base, "")
		u, err = url.Parse(target)
		if err != nil {
			return "", false
		}
	}

	q := u.Query()
	q.Set("error", code)
	u.RawQuery = q.Encode()

	return u.String(), true
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
