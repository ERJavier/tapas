//go:build !darwin && !linux

package ports

import "errors"

func init() {
	defaultLister = &unsupportedLister{}
}

type unsupportedLister struct{}

func (u *unsupportedLister) List() ([]Port, error) {
	return nil, errors.New("TAPAS is supported on macOS and Linux only")
}
