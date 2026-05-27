package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	var (
		dir  string
		port int
	)

	flag.StringVar(&dir, "dir", ".", "directory to serve")
	flag.IntVar(&port, "port", 9245, "port to listen on")
	flag.Parse()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("resolve frontend dir: %v", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		log.Fatalf("stat frontend dir: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("frontend path is not a directory: %s", absDir)
	}

	handler := http.FileServer(http.Dir(absDir))
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("serving frontend from %s at http://%s", absDir, addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("frontend dev server failed: %v", err)
	}
}
