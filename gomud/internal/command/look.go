package command

import (
	"fmt"
	"strings"
)

// LookCommand prints the current room's description, exits, items and players.
func LookCommand(ctx *Context, _ []string) {
	room, err := ctx.World.Room(ctx.Player.CurrentRoomID)
	if err != nil {
		ctx.Player.Send("You are nowhere. How unusual.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\r\n--- %s ---\r\n", room.Name))
	sb.WriteString(room.Description + "\r\n")

	// Exits
	if len(room.Exits) > 0 {
		dirs := make([]string, 0, len(room.Exits))
		for d := range room.Exits {
			dirs = append(dirs, d)
		}
		sb.WriteString(fmt.Sprintf("Exits: %s\r\n", strings.Join(dirs, ", ")))
	} else {
		sb.WriteString("Exits: none\r\n")
	}

	// Items in room
	itemIDs := room.ItemsInRoom()
	for _, id := range itemIDs {
		it, err := ctx.World.Item(id)
		if err == nil {
			sb.WriteString(fmt.Sprintf("  %s is here.\r\n", it.Name))
		}
	}

	// Other players in room
	for _, p := range room.PlayersInRoom() {
		if p.Name != ctx.Player.Name {
			sb.WriteString(fmt.Sprintf("  %s is here.\r\n", p.Name))
		}
	}

	ctx.Player.Send(sb.String())
}
