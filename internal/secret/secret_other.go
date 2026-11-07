//go:build !windows

package secret

import "errors"

func Save(path, value string) error {
	return errors.New("persistent API keys are currently supported only on Windows; use TRENDS_YOUTUBE_API_KEY")
}
func Load(path string) (string, error) { return "", errors.New("use TRENDS_YOUTUBE_API_KEY") }
func Supported() bool                  { return false }
