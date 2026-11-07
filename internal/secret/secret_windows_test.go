//go:build windows

package secret

import (
	"bytes"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
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

func TestExistingDPAPIKeyRemainsReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "youtube.key")
	in := blob([]byte("synthetic-existing-key"))
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		t.Fatal(err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	legacy := append([]byte(nil), unsafe.Slice(out.Data, out.Size)...)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	value, err := Load(path)
	if err != nil || value != "synthetic-existing-key" {
		t.Fatal("existing key is no longer readable")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(legacy, after) {
		t.Fatal("read unexpectedly rewrote an existing key")
	}
}

func TestFailedKeyReplacementPreservesOldKeyAndCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "youtube.key")
	if err := Save(path, "synthetic-old-key"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	name, _ := windows.UTF16PtrFromString(path)
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if err := Save(path, "synthetic-new-key"); err == nil {
		t.Fatal("locked key replacement should fail")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed replacement damaged existing key")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "youtube-key-*.tmp"))
	if len(files) != 0 {
		t.Fatal("temporary key left behind")
	}
	if err := Save(path, ""); err == nil {
		t.Fatal("empty key accepted")
	}
}
