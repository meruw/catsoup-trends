//go:build windows

package secret

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

func blob(b []byte) windows.DataBlob { return windows.DataBlob{Size: uint32(len(b)), Data: &b[0]} }
func unlock(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 16384 {
		return nil, errors.New("invalid protected key")
	}
	in := blob(raw)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, errors.New("Windows could not unlock the saved key")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
func Save(path, value string) error {
	raw := []byte(value)
	if len(raw) == 0 || len(raw) > 128 {
		return errors.New("invalid API key length")
	}
	in := blob(raw)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return errors.New("Windows could not protect the API key")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	encrypted := unsafe.Slice(out.Data, out.Size)
	verified, err := unlock(encrypted)
	if err != nil || !bytes.Equal(verified, raw) {
		return errors.New("could not verify protected API key")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "youtube-key-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(encrypted); err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp.Name(), path)
}
func Load(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 16385))
	if err != nil {
		return "", err
	}
	decoded, err := unlock(raw)
	if err != nil {
		return "", err
	}
	if len(decoded) == 0 || len(decoded) > 128 {
		return "", errors.New("invalid saved API key")
	}
	return string(decoded), nil
}
func Supported() bool { return true }
