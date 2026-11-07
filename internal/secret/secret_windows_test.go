//go:build windows

package secret

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedStorageRoundTripReplacementAndCorruption(t *testing.T) {
	p := filepath.Join(t.TempDir(), "youtube.key")
	for _, value := range []string{"synthetic-first-key", "synthetic-replacement-key"} {
		if err := Save(p, value); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(p)
		if err != nil || bytes.Contains(raw, []byte(value)) {
			t.Fatal("plaintext key or unreadable storage")
		}
		got, err := Load(p)
		if err != nil || got != value {
			t.Fatal("protected storage round trip failed")
		}
	}
	if err := os.WriteFile(p, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Fatal("corrupt key was accepted")
	}
}
