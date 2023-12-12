package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	gssh "github.com/gliderlabs/ssh"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	terminal "golang.org/x/term"
)

type Sshebra struct {
	ListenAddr           string
	Authenticator        func(gssh.Context, gssh.PublicKey) (string, error)
	CmdBuilder           func(ctx context.Context) *cobra.Command
	ServerPrivateKeyPath string
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
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	ctx = context.WithValue(ctx,
		"ssh-close", func() { s.Close() })
	ctx = context.WithValue(ctx,
		"ssh-identity", identity)

	rcmd := b.CmdBuilder(ctx)
	io.WriteString(s, fmt.Sprintf("hello, %s\n", identity))
	term := terminal.NewTerminal(s,
		fmt.Sprintf("%s> ", identity))

	pty, winCh, isPty := s.Pty()
	if isPty {
		_ = pty
		go func() {
			for chInfo := range winCh {
				_ = term.SetSize(chInfo.Width, chInfo.Height)
			}
		}()
	}

	rcmd.SetOut(term)
	rcmd.SetErr(term)
	rcmd.SetIn(s) // i hope that ok
	for _, cc := range rcmd.Commands() {
		cc.SetOut(term)
		cc.SetErr(term)
		cc.SetIn(s)
	}

	for {
		line, err := term.ReadLine()
		if err == io.EOF {
			_, _ = io.WriteString(s, "EOF.\n")
			break
		}
		if err != nil {
			_, _ = io.WriteString(s,
				fmt.Errorf("reading line: %w", err).Error())
			break
		}

		args, err := shlex.Split(line)
		if err != nil {
			io.WriteString(term,
				fmt.Errorf("splitting args: %w", err).Error())
			continue
		}
		rcmd.SetArgs(args)
		io.WriteString(term, "\n")
		err = rcmd.Execute()
		if err != nil {
			io.WriteString(term, err.Error())
		}
		if err == context.Canceled {
			break
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

func main() {
	b := &Sshebra{
		ListenAddr: ":4242",
		CmdBuilder: func(ctx context.Context) *cobra.Command {

			rootCmd := &cobra.Command{Use: "type something.."}
			rootCmd.SetContext(ctx)

			var cmdEcho = &cobra.Command{
				Use:   "echo [string to echo]",
				Short: "Echo anything to the screen",
				Args:  cobra.MinimumNArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					cmd.Printf("echo: %s\n", strings.Join(args, " "))
				},
			}
			cmdEcho.SetContext(ctx)

			var echoTimes int
			var cmdTimes = &cobra.Command{
				Use:   "times [string to echo]",
				Short: "Echo anything to the screen more times",
				Args:  cobra.MinimumNArgs(1),
				Run: func(cmd *cobra.Command, args []string) {
					for i := 0; i < echoTimes; i++ {
						cmd.Printf("time #%d: %s\n", i, strings.Join(args, " "))
					}
				},
			}
			cmdTimes.SetContext(ctx)
			cmdTimes.Flags().IntVarP(&echoTimes,
				"times", "t", 1, "times to echo the input")

			cmdEcho.AddCommand(cmdTimes)
			rootCmd.AddCommand(cmdEcho)

			var exitCmd = &cobra.Command{
				Use:   "exit",
				Short: "terminate ssh session",
				Args:  cobra.NoArgs,
				Run: func(cmd *cobra.Command, args []string) {
					cancel, ok := cmd.Context().Value("ssh-close").(func())
					if !ok {
						cmd.Println("close func not casted")
						return
					}
					cancel()
				},
			}
			exitCmd.SetContext(ctx)
			rootCmd.AddCommand(exitCmd)

			var whoamiCmd = &cobra.Command{
				Use:   "whoami",
				Short: "print user identity",
				Args:  cobra.NoArgs,
				Run: func(cmd *cobra.Command, args []string) {
					identity, ok := cmd.Context().Value("ssh-identity").(string)
					if !ok {
						cmd.Println("user identity not casted")
						return
					}
					cmd.Println(identity)
				},
			}
			whoamiCmd.SetContext(ctx)
			rootCmd.AddCommand(whoamiCmd)

			var shellCmd = &cobra.Command{
				Use:   "shell",
				Short: "run system shell command, ex shell 'ls -hlatr'",
				Args:  cobra.ExactArgs(1),
				Run: func(cmd *cobra.Command, args []string) {

					shargs := []string{"-c",
						fmt.Sprintf("%s", strings.Join(args, ""))}

					ee := exec.CommandContext(cmd.Context(), "sh", shargs...)
					ee.Stdout = cmd.OutOrStderr()
					ee.Stderr = cmd.ErrOrStderr()
					ee.Stdin = cmd.InOrStdin()
					err := ee.Run()
					if err != nil {
						cmd.Println(err.Error())
					}
				},
			}
			shellCmd.SetContext(ctx)
			rootCmd.AddCommand(shellCmd)

			return rootCmd
		},
		Authenticator: func(ctx gssh.Context, key gssh.PublicKey) (string, error) {
			mykey, err := os.ReadFile("./mykey.pub")
			if err != nil {
				return "", errors.New("moew")
			}
			pk, _, _, _, err := gssh.ParseAuthorizedKey(mykey)
			if err != nil {
				return "", errors.New("moew")
			}
			if !bytes.Equal(key.Marshal(), pk.Marshal()) {
				return "", errors.New("moew")
			}
			return "root", nil
		},
		ServerPrivateKeyPath: "./mykey",
	}

	log.Println("starting ssh server ")
	log.Fatal(b.ListenAndServe())
}
