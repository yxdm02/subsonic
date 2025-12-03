package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func Serve(distFS fs.FS, debugNetwork bool, port string) {
	hub := NewHub(debugNetwork)
	go hub.Run()

	mux := http.NewServeMux()

	// API endpoint for WebSocket
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// API endpoint for file uploads
	mux.HandleFunc("/api/upload-wordlist", handleUploadWordlist)

	// Static file serving
	staticFS := http.FS(distFS)
	fileServer := http.FileServer(staticFS)

	htmlContent, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to read index.html from embedded fs: %v", err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		
		if _, err := distFS.Open(cleanPath); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(htmlContent)
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})

	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("http.ListenAndServe: %v", err)
	}
}

func handleUploadWordlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadDir := filepath.Join(".", "wordlists", "temp")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	r.ParseMultipartForm(1 << 30)

	file, _, err := r.FormFile("wordlist")
	if err != nil {
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFileName := uuid.New().String() + ".txt"
	tempFilePath := filepath.Join(uploadDir, tempFileName)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"wordlist_key": "temp/" + tempFileName,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
