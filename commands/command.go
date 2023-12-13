package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	terminal "golang.org/x/term"
)

type Command interface {
	Execute(context.Context, []string) error
}

type ExitCommand struct{}

func (c *ExitCommand) Execute(ctx context.Context, args []string) error {
	return context.Canceled
}

type WhoamiCommand struct{}

func (c *WhoamiCommand) Execute(ctx context.Context, args []string) error {
	term, ok := ctx.Value("terminal").(*terminal.Terminal)
	if !ok {
		return errors.New("error getting ssh terminal output")
	}
	identity, ok := ctx.Value("ssh-identity").(string)
	if !ok {
		return errors.New("error getting ssh identity")
	}
	io.WriteString(term, fmt.Sprintf("iknowyouarebutwhatami: %s\n", identity))
	return nil
}

type FlagCommand struct{}

func (c *FlagCommand) Execute(ctx context.Context, args []string) error {
	flagCmd := flag.NewFlagSet("foo", flag.ExitOnError)
	enableP := flagCmd.Bool("enable", false, "enable")
	nameP := flagCmd.String("name", "", "name")
	err := flagCmd.Parse(args)
	if err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	term, ok := ctx.Value("terminal").(*terminal.Terminal)
	if !ok {
		return errors.New("error getting ssh terminal output")
	}
	io.WriteString(term, fmt.Sprintf("parsed flags:\n"))
	io.WriteString(term, fmt.Sprintf("  - enable: \t%v\n", *enableP))
	io.WriteString(term, fmt.Sprintf("  - name: \t%v\n", *nameP))
	io.WriteString(term, fmt.Sprintf("  - tail: \t%v\n", flagCmd.Args()))

	return nil
}
