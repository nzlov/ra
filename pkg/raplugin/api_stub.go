//go:build !wasip1 && !(js && wasm)

package raplugin

import "errors"

var appsListForTesting []App

func AppsList() ([]App, error) {
	if appsListForTesting != nil {
		return append([]App(nil), appsListForTesting...), nil
	}
	return nil, errors.New("apps.list is only available inside RA plugins")
}

func SetAppsListForTesting(apps []App) {
	appsListForTesting = append([]App(nil), apps...)
}

func ResetAppsListForTesting() {
	appsListForTesting = nil
}
