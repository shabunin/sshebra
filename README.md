# sshebra

ssh-server with custom commands inside.

Inspired by:
* [github.com/shazow/ssh-chat](https://github.com/shazow/ssh-chat)
* [github.com/quackduck/devzat](https://github.com/quackduck/devzat)

## Description

```text
go run .
```

```
ssh doesntmatter@localhost -p 4242 -i ./mykey
...
```

## Features

[github.com/gliderlabs/ssh](https://github.com/gliderlabs/ssh) features:
 - [x] PTY terminal 
 - [x] Password auth
 - [x] Pubkey authentication
 - [x] Subsystem handling

