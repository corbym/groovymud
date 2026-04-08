package command

import (
	"fmt"
	"strings"

	"github.com/corbym/groovymud/gomud/internal/event"
	"github.com/corbym/groovymud/gomud/internal/world"
)

// makeGoCommand returns a handler for "go <direction>".
func makeGoCommand(w *world.World, bus *event.Bus) Handler {
	return func(ctx *Context, args []string) {
		if len(args) == 0 {
			ctx.Player.Send("Go where? Specify a direction.")
			return
		}
		movePlayer(ctx, strings.ToLower(args[0]), bus)
	}
}

// makeDirectionCommand returns a handler for bare direction words (north, s, etc.).
func makeDirectionCommand(dir string, w *world.World, bus *event.Bus) Handler {
	return func(ctx *Context, _ []string) {
		movePlayer(ctx, dir, bus)
	}
}

// movePlayer moves ctx.Player in the given direction.
func movePlayer(ctx *Context, dir string, bus *event.Bus) {
	room, err := ctx.World.Room(ctx.Player.CurrentRoomID)
	if err != nil {
		ctx.Player.Send("You appear to be nowhere.")
		return
	}

	targetID, ok := room.Exits[dir]
	if !ok {
		ctx.Player.Send(fmt.Sprintf("You can't go %s from here.", dir))
		return
	}

	targetRoom, err := ctx.World.Room(targetID)
	if err != nil {
		ctx.Player.Send("That exit leads nowhere — the world has a hole in it.")
		return
	}

	// Departure
	oldRoomID := ctx.Player.CurrentRoomID
	room.RemovePlayer(ctx.Player)
	bus.Publish(event.Event{
		Type:    event.Movement,
		Scope:   event.LocalScope,
		RoomID:  oldRoomID,
		ActorID: ctx.Player.Name,
		Payload: fmt.Sprintf("%s leaves %s.", ctx.Player.Name, dir),
	})

	// Move
	ctx.Player.CurrentRoomID = targetID
	targetRoom.AddPlayer(ctx.Player)
	bus.Publish(event.Event{
		Type:    event.Movement,
		Scope:   event.LocalScope,
		RoomID:  targetID,
		ActorID: ctx.Player.Name,
		Payload: fmt.Sprintf("%s arrives.", ctx.Player.Name),
	})

	// Auto-look
	LookCommand(ctx, nil)
}
