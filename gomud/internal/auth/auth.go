package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/corbym/groovymud/gomud/internal/store"
)

// Player is a lightweight view of the store.Player used by callers.
type Player struct {
	ID            int64
	Name          string
	Role          string
	CurrentRoomID string
}

// Register creates a new player account.
func Register(name, password string) (*Player, error) {
	if name == "" || password == "" {
		return nil, errors.New("name and password must not be empty")
	}
	if store.PlayerExists(name) {
		return nil, fmt.Errorf("player %q already exists", name)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	p, err := store.CreatePlayer(name, string(hash))
	if err != nil {
		return nil, fmt.Errorf("creating player: %w", err)
	}
	return fromStore(p), nil
}

// Login verifies credentials and returns the player.
func Login(name, password string) (*Player, error) {
	p, err := store.FindPlayerByName(name)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(p.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return fromStore(p), nil
}

// PlayerExists reports whether a player with the given name exists.
func PlayerExists(name string) bool {
	return store.PlayerExists(name)
}

func fromStore(p *store.Player) *Player {
	return &Player{
		ID:            p.ID,
		Name:          p.Name,
		Role:          p.Role,
		CurrentRoomID: p.CurrentRoomID,
	}
}
