package main

import (
	"fmt"
	"sort"
)

const (
	envDev        = "dev"
	envStaging    = "staging"
	envProduction = "production"
	appName       = "brandishbot"
	confirmYes    = "yes"
)

// Command interface that all devtool commands must implement
type Command interface {
	Name() string
	Description() string
	Run(args []string) error
}

// Registry manages the available commands
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry
func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

// Get retrieves a command by name
func (r *Registry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// List returns a sorted list of all registered commands
func (r *Registry) List() []Command {
	cmds := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name() < cmds[j].Name()
	})
	return cmds
}

// PrintHelp prints the usage information
func (r *Registry) PrintHelp() {
	fmt.Println("Usage: devtool <command> [args...]")
	fmt.Println("\nAvailable Commands:")

	cmds := r.List()
	maxLen := 0
	for _, cmd := range cmds {
		if len(cmd.Name()) > maxLen {
			maxLen = len(cmd.Name())
		}
	}

	for _, cmd := range cmds {
		padding := maxLen - len(cmd.Name()) + 2
		fmt.Printf("  %s%*s%s\n", cmd.Name(), padding, "", cmd.Description())
	}
}
