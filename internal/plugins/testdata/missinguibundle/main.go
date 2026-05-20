package main

import "github.com/nzlov/ra/pkg/raplugin"

func init() {
	raplugin.Register(raplugin.Plugin{
		Manifest: raplugin.Manifest{ID: "missing-ui", Name: "Missing UI", Version: "0.1.0"},
		Capabilities: []raplugin.Capability{{
			ID:    "main",
			Title: "Main",
			UI:    "/main/index.html",
		}},
	})
}

func main() {}
