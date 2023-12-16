package main

import (
	"bytes"
	"io"
	"log"
	"os"

	gssh "github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/shabunin/sshebra/commands"
	"github.com/shabunin/sshebra/sshebra"
)

// sftpHandler handler for SFTP subsystem
func sftpHandler(sess gssh.Session) {
	// from
	// https://github.com/gliderlabs/ssh/blob/master/_examples/ssh-sftpserver/sftp.go
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		log.Println("sftp client exited session.")
	} else if err != nil {
		log.Println("sftp server completed with error:", err)
	}
	log.Println("bye from sftp handler")
}

func pubkeyAuth(ctx gssh.Context, key gssh.PublicKey) bool {
	mykey, err := os.ReadFile("./mykey.pub")
	if err != nil {
		log.Printf("reading file: %w\n", err)
		return false
	}
	pk, _, _, _, err := gssh.ParseAuthorizedKey(mykey)
	if err != nil {
		log.Printf("parse auth key: %\n", err)
		return false
	}
	if !bytes.Equal(key.Marshal(), pk.Marshal()) {
		return false
	}
	return true
}
func passwordAuth(ctx gssh.Context, password string) bool {
	return password == "parole"
}
func main() {
	b := &sshebra.Sshebra{}
	b.RegisterCommand("whoami", &commands.WhoamiCommand{})
	b.RegisterCommand("exit", &commands.ExitCommand{})
	b.RegisterCommand("flag", &commands.FlagCommand{})

	s := &gssh.Server{
		Addr: ":4242",
		Handler: b.SessionHandler,
		SubsystemHandlers: map[string]gssh.SubsystemHandler{
			"sftp": sftpHandler,
		},
		PublicKeyHandler: pubkeyAuth,
		PasswordHandler: passwordAuth,
	}
	gssh.HostKeyFile("./mykey")(s)

	log.Println("starting ssh server ")
	log.Fatal(s.ListenAndServe())
}
