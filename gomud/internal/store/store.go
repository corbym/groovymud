package store

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	globalDB *sql.DB
	globalMu sync.RWMutex
)

// Player represents a player record in the database.
type Player struct {
	ID            int64
	Name          string
	PasswordHash  string
	Role          string
	CurrentRoomID string
}

// Open opens (or creates) the SQLite database at dsn and runs migrations.
// Use dsn = "file::memory:?cache=shared" for in-memory databases in tests.
func Open(dsn string) error {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("store.Open: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return fmt.Errorf("store.Open migrate: %w", err)
	}
	globalMu.Lock()
	globalDB = db
	globalMu.Unlock()
	return nil
}

// Close closes the global database.
func Close() {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalDB != nil {
		_ = globalDB.Close()
		globalDB = nil
	}
}

func db() *sql.DB {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalDB
}

// migrate runs the schema creation DDL.
func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			id              INTEGER PRIMARY KEY,
			name            TEXT UNIQUE NOT NULL,
			password_hash   TEXT NOT NULL,
			role            TEXT NOT NULL DEFAULT 'player',
			current_room_id TEXT NOT NULL DEFAULT 'town:centre',
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS player_inventory (
			player_id INTEGER NOT NULL,
			item_id   TEXT    NOT NULL
		);
	`)
	return err
}

// CreatePlayer inserts a new player and returns it.
func CreatePlayer(name, passwordHash string) (*Player, error) {
	res, err := db().Exec(
		`INSERT INTO players (name, password_hash) VALUES (?, ?)`,
		name, passwordHash,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Player{
		ID:            id,
		Name:          name,
		PasswordHash:  passwordHash,
		Role:          "player",
		CurrentRoomID: "town:centre",
	}, nil
}

// FindPlayerByName retrieves a player by name (case-insensitive).
func FindPlayerByName(name string) (*Player, error) {
	row := db().QueryRow(
		`SELECT id, name, password_hash, role, current_room_id FROM players WHERE name = ? COLLATE NOCASE`,
		name,
	)
	p := &Player{}
	if err := row.Scan(&p.ID, &p.Name, &p.PasswordHash, &p.Role, &p.CurrentRoomID); err != nil {
		return nil, err
	}
	return p, nil
}

// PlayerExists returns true if a player with the given name exists.
func PlayerExists(name string) bool {
	_, err := FindPlayerByName(name)
	return err == nil
}

// UpdateRoomID updates the player's current room.
func UpdateRoomID(playerID int64, roomID string) error {
	_, err := db().Exec(`UPDATE players SET current_room_id = ? WHERE id = ?`, roomID, playerID)
	return err
}

// AddItem adds an item to a player's inventory.
func AddItem(playerID int64, itemID string) error {
	_, err := db().Exec(`INSERT INTO player_inventory (player_id, item_id) VALUES (?, ?)`, playerID, itemID)
	return err
}

// RemoveItem removes one copy of an item from a player's inventory.
func RemoveItem(playerID int64, itemID string) error {
	_, err := db().Exec(
		`DELETE FROM player_inventory WHERE rowid = (
			SELECT rowid FROM player_inventory WHERE player_id = ? AND item_id = ? LIMIT 1
		)`,
		playerID, itemID,
	)
	return err
}

// GetInventory returns all item IDs in a player's inventory.
func GetInventory(playerID int64) ([]string, error) {
	rows, err := db().Query(`SELECT item_id FROM player_inventory WHERE player_id = ?`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}
