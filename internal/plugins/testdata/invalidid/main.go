package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "../escape", Name: "Invalid", Version: "0.1.0"},
	})
}

func main() {}
