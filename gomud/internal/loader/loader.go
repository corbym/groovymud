package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/corbym/groovymud/gomud/internal/world"
)

// tomlFile is the top-level structure of a world TOML file.
type tomlFile struct {
	Rooms []tomlRoom `toml:"room"`
	Items []tomlItem `toml:"item"`
}

type tomlRoom struct {
	ID               string            `toml:"id"`
	Name             string            `toml:"name"`
	Description      string            `toml:"description"`
	ShortDescription string            `toml:"short_description"`
	Exits            map[string]string `toml:"exits"`
	Items            []string          `toml:"items"`
}

type tomlItem struct {
	ID         string   `toml:"id"`
	Name       string   `toml:"name"`
	ShortNames []string `toml:"short_names"`
}

// LoadDir loads all *.toml files from dir into the world.
func LoadDir(w *world.World, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("loader.LoadDir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".toml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := LoadFile(w, path); err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}
	}
	return nil
}

// LoadFile loads a single TOML file into the world.
func LoadFile(w *world.World, path string) error {
	var tf tomlFile
	if _, err := toml.DecodeFile(path, &tf); err != nil {
		return fmt.Errorf("loader.LoadFile %s: %w", path, err)
	}

	for _, ti := range tf.Items {
		it := &world.Item{
			MudObject:  world.MudObject{ID: ti.ID, Name: ti.Name},
			ShortNames: ti.ShortNames,
		}
		w.AddItem(it)
	}

	for _, tr := range tf.Rooms {
		exits := tr.Exits
		if exits == nil {
			exits = make(map[string]string)
		}
		r := &world.Room{
			MudObject:        world.MudObject{ID: tr.ID, Name: tr.Name, Description: tr.Description},
			ShortDescription: tr.ShortDescription,
			Exits:            exits,
			ItemIDs:          append([]string(nil), tr.Items...),
		}
		w.AddRoom(r)
	}
	return nil
}
