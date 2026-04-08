package command

import (
	"fmt"
	"strings"
)

// GetCommand handles "get <item>".
func GetCommand(ctx *Context, args []string) {
	if len(args) == 0 {
		ctx.Player.Send("Get what?")
		return
	}
	name := strings.Join(args, " ")

	room, err := ctx.World.Room(ctx.Player.CurrentRoomID)
	if err != nil {
		ctx.Player.Send("You are nowhere.")
		return
	}

	item := ctx.World.FindItemByShortName(room.ItemsInRoom(), name)
	if item == nil {
		ctx.Player.Send(fmt.Sprintf("You don't see %q here.", name))
		return
	}

	if !room.RemoveItem(item.ID) {
		ctx.Player.Send("It seems to have vanished.")
		return
	}
	ctx.Player.Inventory = append(ctx.Player.Inventory, item.ID)
	ctx.Player.Send(fmt.Sprintf("You pick up %s.", item.Name))
}

// DropCommand handles "drop <item>".
func DropCommand(ctx *Context, args []string) {
	if len(args) == 0 {
		ctx.Player.Send("Drop what?")
		return
	}
	name := strings.Join(args, " ")

	item := ctx.World.FindItemByShortName(ctx.Player.Inventory, name)
	if item == nil {
		ctx.Player.Send(fmt.Sprintf("You don't have %q.", name))
		return
	}

	// Remove from inventory
	for i, id := range ctx.Player.Inventory {
		if id == item.ID {
			ctx.Player.Inventory = append(ctx.Player.Inventory[:i], ctx.Player.Inventory[i+1:]...)
			break
		}
	}

	room, err := ctx.World.Room(ctx.Player.CurrentRoomID)
	if err != nil {
		ctx.Player.Send("You are nowhere.")
		return
	}
	room.AddItem(item.ID)
	ctx.Player.Send(fmt.Sprintf("You drop %s.", item.Name))
}

// InventoryCommand handles "inventory" / "i".
func InventoryCommand(ctx *Context, _ []string) {
	if len(ctx.Player.Inventory) == 0 {
		ctx.Player.Send("You are not carrying anything.")
		return
	}
	ctx.Player.Send("You are carrying:")
	for _, id := range ctx.Player.Inventory {
		it, err := ctx.World.Item(id)
		if err != nil {
			ctx.Player.Send(fmt.Sprintf("  <unknown item %s>", id))
			continue
		}
		ctx.Player.Send(fmt.Sprintf("  %s", it.Name))
	}
}

// PutCommand handles "put <item> in <container>".
func PutCommand(ctx *Context, args []string) {
	// parse: put <item> in <container>
	line := strings.Join(args, " ")
	idx := strings.Index(strings.ToLower(line), " in ")
	if idx < 0 {
		ctx.Player.Send("Syntax: put <item> in <container>")
		return
	}
	itemName := strings.TrimSpace(line[:idx])
	containerName := strings.TrimSpace(line[idx+4:])

	item := ctx.World.FindItemByShortName(ctx.Player.Inventory, itemName)
	if item == nil {
		ctx.Player.Send(fmt.Sprintf("You don't have %q.", itemName))
		return
	}

	// Find the container in the room or inventory.
	room, err := ctx.World.Room(ctx.Player.CurrentRoomID)
	if err != nil {
		ctx.Player.Send("You are nowhere.")
		return
	}

	allIDs := append(room.ItemsInRoom(), ctx.Player.Inventory...)
	container := ctx.World.FindItemByShortName(allIDs, containerName)
	if container == nil {
		ctx.Player.Send(fmt.Sprintf("You don't see %q here.", containerName))
		return
	}

	// Remove item from player inventory.
	for i, id := range ctx.Player.Inventory {
		if id == item.ID {
			ctx.Player.Inventory = append(ctx.Player.Inventory[:i], ctx.Player.Inventory[i+1:]...)
			break
		}
	}
	ctx.Player.Send(fmt.Sprintf("You put %s in %s.", item.Name, container.Name))
}
