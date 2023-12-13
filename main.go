package main

import (
	"bytes"
	"errors"
	"log"
	"os"

	gssh "github.com/gliderlabs/ssh"
	"github.com/shabunin/sshebra/commands"
	"github.com/shabunin/sshebra/sshebra"
)

func main() {
	b := &sshebra.Sshebra{
		ListenAddr: ":4242",
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
	b.RegisterCommand("whoami", &commands.WhoamiCommand{})
	b.RegisterCommand("exit", &commands.ExitCommand{})
	b.RegisterCommand("flag", &commands.FlagCommand{})

	log.Println("starting ssh server ")
	log.Fatal(b.ListenAndServe())
}
