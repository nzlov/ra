package raplugin

import (
	"io/fs"
	"path"
	"strings"
)

func MustAssets(fsys fs.FS, root string) map[string][]byte {
	assets := map[string][]byte{}
	err := fs.WalkDir(fsys, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		assetPath := strings.TrimPrefix(path.Clean(strings.TrimPrefix(name, root)), ".")
		if !strings.HasPrefix(assetPath, "/") {
			assetPath = "/" + assetPath
		}
		assets[assetPath] = raw
		return nil
	})
	if err != nil {
		panic(err)
	}
	return assets
}
