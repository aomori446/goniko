package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
)

type command struct {
	name  string
	value any
}

type Node struct {
	Name         string
	Description  string
	FlagSet      *flag.FlagSet
	mustCommands []command
	subNodes     []*Node
	Action       func()
}

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func New(name string, description string) *Node {
	base := filepath.Base(name)
	fs := flag.NewFlagSet(base, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return &Node{
		Name:         base,
		Description:  description,
		FlagSet:      fs,
		mustCommands: make([]command, 0),
		subNodes:     make([]*Node, 0),
	}
}

func (n *Node) AddSubNode(sub *Node) {
	n.subNodes = append(n.subNodes, sub)
}

func (n *Node) String(name, defaultValue, usage string, must bool) *string {
	cmd := n.FlagSet.String(name, defaultValue, usage)
	if must {
		n.mustCommands = append(n.mustCommands, command{name: name, value: cmd})
	}
	return cmd
}

func (n *Node) Bool(name string, defaultValue bool, usage string, must bool) *bool {
	cmd := n.FlagSet.Bool(name, defaultValue, usage)
	if must {
		n.mustCommands = append(n.mustCommands, command{name: name, value: cmd})
	}
	return cmd
}

func (n *Node) Int(name string, defaultValue int, usage string, must bool) *int {
	cmd := n.FlagSet.Int(name, defaultValue, usage)
	if must {
		n.mustCommands = append(n.mustCommands, command{name: name, value: cmd})
	}
	return cmd
}

func (n *Node) Float64(name string, defaultValue float64, usage string, must bool) *float64 {
	cmd := n.FlagSet.Float64(name, defaultValue, usage)
	if must {
		n.mustCommands = append(n.mustCommands, command{name: name, value: cmd})
	}
	return cmd
}

func (n *Node) Parse(args []string) {
	if len(args) < 1 {
		n.usageAndExit()
	}
	for _, sub := range n.subNodes {
		if args[0] == sub.Name {
			if err := sub.FlagSet.Parse(args[1:]); err != nil {
				sub.usageAndExit()
			}
			sub.checkMust()
			if sub.Action != nil {
				sub.Action()
			}
			return
		}
	}
	if err := n.FlagSet.Parse(args); err != nil {
		n.usageAndExit()
	}
	n.checkMust()
	if n.Action != nil {
		n.Action()
	}
}

func (n *Node) checkMust() {
	for _, mc := range n.mustCommands {
		rv := reflect.ValueOf(mc.value).Elem()
		if rv.IsZero() {
			logger.With(
				slog.String("flag", "-"+mc.name),
				slog.String("example", fmt.Sprintf("-%s=<value> or -%s <value>", mc.name, mc.name)),
			).Error("Missing required flag")
			os.Exit(1)
		}
	}
}

func (n *Node) usageAndExit() {
	logger.Error("Invalid usage",
		"command", n.Name,
		"hint", fmt.Sprintf("Usage: %s [subcommand] [options]", n.Name),
	)
	fmt.Fprintf(os.Stderr, "Usage: %s [subcommand] [options]\n\n", n.Name)
	if len(n.subNodes) > 0 {
		fmt.Fprintln(os.Stderr, "Available subcommands:")
		for _, sub := range n.subNodes {
			desc := ""
			if sub.Description != "" {
				desc = " - " + sub.Description
			}
			fmt.Fprintf(os.Stderr, "  %s%s\n", sub.Name, desc)
		}
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr, "Options:")
	n.FlagSet.PrintDefaults()
	os.Exit(1)
}
