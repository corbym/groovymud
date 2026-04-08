package command

import (
	"fmt"
	"strings"

	"github.com/corbym/groovymud/gomud/internal/event"
	"github.com/corbym/groovymud/gomud/internal/world"
)

// Context is passed to every command handler.
type Context struct {
	Player *world.Player
	World  *world.World
	Bus    *event.Bus
}

// Handler is a function that handles one parsed command.
type Handler func(ctx *Context, args []string)

// Registry maps command names to handlers.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds a handler under one or more names/aliases.
func (r *Registry) Register(handler Handler, names ...string) {
	for _, n := range names {
		r.handlers[strings.ToLower(n)] = handler
	}
}

// Dispatch parses a raw input line and calls the matching handler.
// Returns false if the command was not recognised.
func (r *Registry) Dispatch(ctx *Context, line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return true
	}
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	h, ok := r.handlers[cmd]
	if !ok {
		ctx.Player.Send(fmt.Sprintf("Unknown command: %q. Type 'help' for a list.", cmd))
		return false
	}
	h(ctx, args)
	return true
}

// RegisterBuiltins registers all built-in commands on the registry.
func RegisterBuiltins(r *Registry, w *world.World, bus *event.Bus) {
	r.Register(LookCommand, "look", "l")
	r.Register(makeGoCommand(w, bus), "go")
	r.Register(makeDirectionCommand("north", w, bus), "north", "n")
	r.Register(makeDirectionCommand("south", w, bus), "south", "s")
	r.Register(makeDirectionCommand("east", w, bus), "east", "e")
	r.Register(makeDirectionCommand("west", w, bus), "west", "w")
	r.Register(makeDirectionCommand("up", w, bus), "up", "u")
	r.Register(makeDirectionCommand("down", w, bus), "down", "d")
	r.Register(GetCommand, "get", "take")
	r.Register(DropCommand, "drop")
	r.Register(InventoryCommand, "inventory", "i")
	r.Register(PutCommand, "put")
}
