package sshebra

import (
	"context"
	"fmt"
	"io"

	gssh "github.com/gliderlabs/ssh"
	"github.com/google/shlex"
	"github.com/shabunin/sshebra/commands"
	terminal "golang.org/x/term"
)

type Sshebra struct {
	ListenAddr           string
	ServerPrivateKeyPath string
	Authenticator        func(gssh.Context, gssh.PublicKey) (string, error)

	cmds map[string]commands.Command
}

func (b *Sshebra) handler(s gssh.Session) {
	defer s.Close()
	if s.RawCommand() != "" {
		io.WriteString(s, "raw commands are not supported")
		return
	}
	identity, ok := s.Context().Value("ssh-identity").(string)
	if !ok {
		return
	}

	io.WriteString(s, fmt.Sprintf("hello, %s\n", identity))

	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	ctx = context.WithValue(ctx,
		"ssh-close", func() { s.Close() })
	ctx = context.WithValue(ctx,
		"ssh-identity", identity)

	term := terminal.NewTerminal(s,
		fmt.Sprintf("%s> ", identity))

	ctx = context.WithValue(ctx,
		"terminal", term)

	pty, winCh, isPty := s.Pty()
	if isPty {
		_ = pty
		go func() {
			for chInfo := range winCh {
				_ = term.SetSize(chInfo.Width, chInfo.Height)
			}
		}()
	}

	for {
		line, err := term.ReadLine()
		if err == io.EOF {
			_, _ = io.WriteString(s, "EOF.\n")
			break
		}
		if err != nil {
			_, _ = io.WriteString(s,
				fmt.Errorf("reading line: %w\n", err).Error())
			break
		}

		args, err := shlex.Split(line)
		if err != nil || len(args) == 0 {
			io.WriteString(term,
				fmt.Errorf("splitting args: %w\n", err).Error())
			continue
		}
		cmdName := args[0]
		args = args[1:]

		cmd, ok := b.cmds[cmdName]
		if !ok {
			io.WriteString(term,
				fmt.Sprintf("unknown command %s\n", cmdName))
			continue
		}
		err = cmd.Execute(ctx, args)
		io.WriteString(term, "\n")
		if err == context.Canceled {
			break
		}
		if err != nil {
			io.WriteString(term,
				fmt.Errorf("command returned error: %w", err).Error())
		}
	}
}

func (b *Sshebra) authHandler(ctx gssh.Context, key gssh.PublicKey) bool {
	identity, err := b.Authenticator(ctx, key)
	if err != nil {
		return false
	}
	ctx.SetValue("ssh-identity", identity)
	return true
}

func (b *Sshebra) ListenAndServe() error {
	return gssh.ListenAndServe(
		":4242",
		b.handler,
		gssh.HostKeyFile(b.ServerPrivateKeyPath),
		gssh.PublicKeyAuth(b.authHandler),
	)
}

func (b *Sshebra) RegisterCommand(name string, cmd commands.Command) {
	if b.cmds == nil {
		b.cmds = make(map[string]commands.Command)
	}
	b.cmds[name] = cmd
}
