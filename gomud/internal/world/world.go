package world

import (
	"fmt"
	"sync"
)

// MudObject is the base type for anything in the world.
type MudObject struct {
	ID          string
	Name        string
	Description string
}

// Exit describes a directional link from one room to another.
type Exit struct {
	Direction string
	TargetID  string
}

// Item is a tangible object that can exist in rooms or inventories.
type Item struct {
	MudObject
	ShortNames []string
}

// Room is a location in the world.
type Room struct {
	MudObject
	ShortDescription string
	Exits            map[string]string // direction -> room ID
	ItemIDs          []string          // items currently in the room

	mu      sync.Mutex
	Players []*Player // players currently in the room
}

// AddPlayer adds a player to the room (thread-safe).
func (r *Room) AddPlayer(p *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players = append(r.Players, p)
}

// RemovePlayer removes a player from the room (thread-safe).
func (r *Room) RemovePlayer(p *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, pl := range r.Players {
		if pl == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			return
		}
	}
}

// PlayersInRoom returns a snapshot of players in the room.
func (r *Room) PlayersInRoom() []*Player {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Player, len(r.Players))
	copy(out, r.Players)
	return out
}

// AddItem adds an item ID to the room.
func (r *Room) AddItem(itemID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ItemIDs = append(r.ItemIDs, itemID)
}

// RemoveItem removes one copy of an item ID from the room.
func (r *Room) RemoveItem(itemID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, id := range r.ItemIDs {
		if id == itemID {
			r.ItemIDs = append(r.ItemIDs[:i], r.ItemIDs[i+1:]...)
			return true
		}
	}
	return false
}

// ItemsInRoom returns a snapshot of item IDs in the room.
func (r *Room) ItemsInRoom() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.ItemIDs))
	copy(out, r.ItemIDs)
	return out
}

// Player represents an online player in the world.
type Player struct {
	ID            int64
	Name          string
	CurrentRoomID string
	Inventory     []string // item IDs
	mu            sync.Mutex

	// WriteFunc is called to send text to the player's client.
	WriteFunc func(msg string)
}

// Send sends a message to the player.
func (p *Player) Send(msg string) {
	p.mu.Lock()
	fn := p.WriteFunc
	p.mu.Unlock()
	if fn != nil {
		fn(msg)
	}
}

// World holds all loaded game data.
type World struct {
	mu    sync.RWMutex
	Rooms map[string]*Room
	Items map[string]*Item
}

// New creates an empty World.
func New() *World {
	return &World{
		Rooms: make(map[string]*Room),
		Items: make(map[string]*Item),
	}
}

// Room returns a room by ID.
func (w *World) Room(id string) (*Room, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	r, ok := w.Rooms[id]
	if !ok {
		return nil, fmt.Errorf("room %q not found", id)
	}
	return r, nil
}

// Item returns an item by ID.
func (w *World) Item(id string) (*Item, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	it, ok := w.Items[id]
	if !ok {
		return nil, fmt.Errorf("item %q not found", id)
	}
	return it, nil
}

// AddRoom adds a room to the world.
func (w *World) AddRoom(r *Room) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Rooms[r.ID] = r
}

// AddItem adds an item to the world registry.
func (w *World) AddItem(it *Item) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Items[it.ID] = it
}

// FindItemByShortName finds an item in a list of IDs matching a short name.
func (w *World) FindItemByShortName(itemIDs []string, name string) *Item {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, id := range itemIDs {
		it, ok := w.Items[id]
		if !ok {
			continue
		}
		if matchesShortName(it, name) {
			return it
		}
	}
	return nil
}

func matchesShortName(it *Item, name string) bool {
	if it.Name == name {
		return true
	}
	for _, sn := range it.ShortNames {
		if sn == name {
			return true
		}
	}
	return false
}
