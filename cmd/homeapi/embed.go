package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed frontend_build/*
var embeddedFrontend embed.FS

func getEmbeddedFrontend() fs.FS {
	sub, err := fs.Sub(embeddedFrontend, "frontend_build")
	if err != nil {
		log.Printf("Warning: embedded frontend not available: %v", err)
		return nil
	}
	return sub
}
