package webhook

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "webhooks.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// entry builds a webhooks.json with a single webhook. The arguments are raw JSON values.
func entry(name, url, method, events string) string {
	return fmt.Sprintf(`[{"name":%s,"url":%s,"method":%s,"events":%s}]`, name, url, method, events)
}

func TestLoad(t *testing.T) {
	path := writeFile(t, entry(`"crm"`, `"https://crm.example.com/hook"`, `"post"`, `["success","error"]`))

	got, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []Webhook{{
		Name:   "crm",
		URL:    "https://crm.example.com/hook",
		Method: "POST",
		Events: []Event{Success, Error},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "webhooks.json"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected no webhooks, got %+v", got)
	}
}

func TestLoadEmptyList(t *testing.T) {
	got, err := Load(writeFile(t, `[]`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no webhooks, got %+v", got)
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	// A directory exists but can't be read as a file.
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("expected an error")
	}
}

func TestLoadRejectsInvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed json", content: `[{`},
		{name: "not a list", content: `{"name":"crm"}`},
		{name: "unknown field", content: `[{"name":"crm","url":"https://crm.example.com","method":"POST","events":["success"],"headers":{}}]`},
		{name: "empty name", content: entry(`""`, `"https://crm.example.com"`, `"POST"`, `["success"]`)},
		{name: "blank name", content: entry(`"  "`, `"https://crm.example.com"`, `"POST"`, `["success"]`)},
		{name: "empty url", content: entry(`"crm"`, `""`, `"POST"`, `["success"]`)},
		{name: "relative url", content: entry(`"crm"`, `"/hook"`, `"POST"`, `["success"]`)},
		{name: "unsupported scheme", content: entry(`"crm"`, `"ftp://crm.example.com"`, `"POST"`, `["success"]`)},
		{name: "url without host", content: entry(`"crm"`, `"https://"`, `"POST"`, `["success"]`)},
		{name: "empty method", content: entry(`"crm"`, `"https://crm.example.com"`, `""`, `["success"]`)},
		{name: "unsupported method", content: entry(`"crm"`, `"https://crm.example.com"`, `"GET"`, `["success"]`)},
		{name: "missing events", content: entry(`"crm"`, `"https://crm.example.com"`, `"POST"`, `null`)},
		{name: "empty events", content: entry(`"crm"`, `"https://crm.example.com"`, `"POST"`, `[]`)},
		{name: "unknown event", content: entry(`"crm"`, `"https://crm.example.com"`, `"POST"`, `["success","done"]`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(writeFile(t, tt.content))
			if err == nil {
				t.Errorf("expected an error, got %+v", got)
			}
		})
	}
}
