package enbw

import "testing"

func TestExtractKey(t *testing.T) {
	html := `<script>window.cfg = { serviceUrl: "https://x", apimSubscriptionKey: "d4954e8b2e444fc89a89a463788c0a72", foo: 1 };</script>`
	got, err := ExtractKey(html)
	if err != nil {
		t.Fatal(err)
	}
	if got != "d4954e8b2e444fc89a89a463788c0a72" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractKeyMissing(t *testing.T) {
	if _, err := ExtractKey("<html>no key here</html>"); err == nil {
		t.Fatal("expected error when key absent")
	}
}

func TestKeyManagerSeed(t *testing.T) {
	m := NewKeyManager("http://example.invalid", "seed-key", nil)
	if m.Key() != "seed-key" {
		t.Fatalf("seed not stored, got %q", m.Key())
	}
}
