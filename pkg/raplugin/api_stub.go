//go:build !wasip1 && !(js && wasm)

package raplugin

import "errors"

func AppsList() ([]App, error) {
	return nil, errors.New("apps.list is only available inside RA plugins")
}
