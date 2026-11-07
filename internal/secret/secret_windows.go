//go:build windows

package secret

import (
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

func blob(b []byte) windows.DataBlob { return windows.DataBlob{Size: uint32(len(b)), Data: &b[0]} }
func Save(path, value string) error {
	raw := []byte(value)
	in := blob(raw)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	encrypted := unsafe.Slice(out.Data, out.Size)
	if err := os.WriteFile(path+".tmp", encrypted, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
func Load(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", os.ErrInvalid
	}
	in := blob(raw)
	var out windows.DataBlob
	if err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return string(unsafe.Slice(out.Data, out.Size)), nil
}
func Supported() bool { return true }
