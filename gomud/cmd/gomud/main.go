package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/corbym/groovymud/gomud/internal/engine"
	"github.com/corbym/groovymud/gomud/internal/loader"
	"github.com/corbym/groovymud/gomud/internal/store"
	"github.com/corbym/groovymud/gomud/internal/world"
)

func main() {
	port := os.Getenv("MUD_PORT")
	if port == "" {
		port = "2222"
	}

	// Open the SQLite database.
	dbPath := os.Getenv("MUD_DB")
	if dbPath == "" {
		dbPath = "gomud.db"
	}
	if err := store.Open(dbPath); err != nil {
		log.Fatalf("opening db: %v", err)
	}
	defer store.Close()

	// Load world data.
	w := world.New()
	worldDir := os.Getenv("MUD_WORLD_DIR")
	if worldDir == "" {
		// Default: <repo>/gomud/world relative to the binary's source.
		_, filename, _, _ := runtime.Caller(0)
		worldDir = filepath.Join(filepath.Dir(filename), "..", "..", "world")
	}
	if err := loader.LoadDir(w, worldDir); err != nil {
		log.Printf("warning: could not load world dir %s: %v", worldDir, err)
	}

	eng := engine.New(w)
	if err := eng.Server.Listen(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("GroovyMud listening on :%s", port)
	eng.Server.Serve()
}
