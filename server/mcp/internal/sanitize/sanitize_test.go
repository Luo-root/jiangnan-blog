package sanitize

import "testing"

func TestFindEmptyPatterns(t *testing.T) {
	if hits := Find(nil, "password: secret", "token=abc"); hits != nil {
		t.Fatalf("empty patterns should skip, got %v", hits)
	}
	if hits := Find([]string{}, "password: secret"); hits != nil {
		t.Fatalf("[] should skip, got %v", hits)
	}
}

func TestFindWarningOnly(t *testing.T) {
	hits := Find([]string{`-----BEGIN [A-Z ]*PRIVATE KEY-----`}, "hello\n-----BEGIN RSA PRIVATE KEY-----\nend")
	if len(hits) != 1 {
		t.Fatalf("hits = %v", hits)
	}
	if hits[0] != "-----BEGIN RSA PRIVATE KEY-----" {
		t.Fatalf("hit = %q", hits[0])
	}
}
