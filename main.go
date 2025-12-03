package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"subsonic/internal/server"
)

//go:embed all:frontend/dist
var embeddedFiles embed.FS

func main() {
	debugNetwork := flag.Bool("debug-network", false, "Enable detailed network error logging for DNS resolution.")
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	// We must create a sub-filesystem that starts from the 'frontend/dist' directory.
	distFS, err := fs.Sub(embeddedFiles, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to create sub-filesystem: %v", err)
	}
	server.Serve(distFS, *debugNetwork, *port)
}
