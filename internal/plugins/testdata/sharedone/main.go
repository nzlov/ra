package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "shared", Name: "Shared", Version: "0.1.0"},
	})
}

func main() {}
