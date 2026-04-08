package engine

import (
	"fmt"
	"strings"
	"sync"

	"github.com/corbym/groovymud/gomud/internal/command"
	"github.com/corbym/groovymud/gomud/internal/event"
	mudnet "github.com/corbym/groovymud/gomud/internal/net"
	"github.com/corbym/groovymud/gomud/internal/store"
	"github.com/corbym/groovymud/gomud/internal/world"
)

// Engine wires together the server, world and command registry.
type Engine struct {
	Server   *mudnet.Server
	World    *world.World
	Bus      *event.Bus
	Registry *command.Registry

	mu      sync.RWMutex
	players map[string]*world.Player // keyed by lower-case name
}

// New creates a new Engine.
func New(w *world.World) *Engine {
	srv := mudnet.NewServer()
	bus := event.NewBus()
	reg := command.NewRegistry()
	command.RegisterBuiltins(reg, w, bus)

	e := &Engine{
		Server:   srv,
		World:    w,
		Bus:      bus,
		Registry: reg,
		players:  make(map[string]*world.Player),
	}

	srv.OnLogin = e.onLogin
	srv.OnCommand = e.onCommand

	// Broadcast local-scope events (movement, messages) to room members.
	bus.Subscribe(e.handleEvent)

	return e
}

// onLogin is called by the server after a session authenticates.
func (e *Engine) onLogin(s *mudnet.Session) {
	name := s.Name()

	startRoom := "town:centre"
	if sp, err := store.FindPlayerByName(name); err == nil {
		startRoom = sp.CurrentRoomID
	}

	wp := &world.Player{
		Name:          name,
		CurrentRoomID: startRoom,
		WriteFunc: func(msg string) {
			_ = s.WriteLine(msg)
		},
	}
	e.mu.Lock()
	e.players[strings.ToLower(name)] = wp
	e.mu.Unlock()

	// Place player in starting room.
	room, err2 := e.World.Room(wp.CurrentRoomID)
	if err2 == nil {
		room.AddPlayer(wp)
	}

	// Auto-look on login.
	ctx := &command.Context{Player: wp, World: e.World, Bus: e.Bus}
	command.LookCommand(ctx, nil)
}

// onCommand dispatches a line of input from a session.
func (e *Engine) onCommand(s *mudnet.Session, line string) {
	name := s.Name()
	e.mu.RLock()
	wp := e.players[strings.ToLower(name)]
	e.mu.RUnlock()
	if wp == nil {
		return
	}
	ctx := &command.Context{Player: wp, World: e.World, Bus: e.Bus}
	e.Registry.Dispatch(ctx, line)
}

// handleEvent routes local-scope events to players in the relevant room.
func (e *Engine) handleEvent(ev event.Event) {
	if ev.Scope != event.LocalScope {
		return
	}
	msg, ok := ev.Payload.(string)
	if !ok {
		return
	}
	room, err := e.World.Room(ev.RoomID)
	if err != nil {
		return
	}
	for _, p := range room.PlayersInRoom() {
		if p.Name == ev.ActorID {
			continue // don't send departure/arrival messages to the actor
		}
		p.Send(fmt.Sprintf("\r\n%s\r\n> ", msg))
	}
}

// RemovePlayer removes a player from the engine (called on disconnect).
func (e *Engine) RemovePlayer(name string) {
	e.mu.Lock()
	wp := e.players[strings.ToLower(name)]
	delete(e.players, strings.ToLower(name))
	e.mu.Unlock()
	if wp == nil {
		return
	}
	room, err := e.World.Room(wp.CurrentRoomID)
	if err == nil {
		room.RemovePlayer(wp)
	}
}
