package main

import (
	"os"
	"net"
	"fmt"
	"bufio"
	"unicode"
	"strings"
	"encoding/json"
)

func main() {
	var cfg config
	var irc ircConn

	fmt.Println("Loading", configPath)
	b, e := os.ReadFile(configPath)
	if e != nil {
		panic(e)
	}
	if e = json.Unmarshal(b, &cfg); e != nil {
		panic(e)
	}

	fmt.Println("Connecting to", cfg.IRC.Addr)
	irc.Conn, e = net.Dial("tcp", cfg.IRC.Addr)
	if e != nil {
		panic(e)
	}
	irc.scanner = bufio.NewScanner(irc.Conn)
	defer irc.Close()

	fmt.Println("Authenticating as", cfg.IRC.User)
	fmt.Fprintf(irc, "PASS %v\nNICK %v\n", cfg.IRC.Pass, cfg.IRC.User)
	for {
		m, e := irc.get()
		if e != nil {
			panic(e)
		}
		if m.cmd == "001" {
			break
		}
		fmt.Printf("%v: %v\n", m.src, strings.Join(m.args, " "))
	}

	for m, e := irc.get(); e == nil; m, e = irc.get() {
		if m.cmd == "PING" {
			fmt.Fprintf(irc, "PONG\n")
		} else if m.cmd != "QUIT" {
			fmt.Println(m)
		}
	}
}

const (
	configPath = "config.json"
)

type config struct {
	IRC struct {
		Addr string `json:"address"`
		User string `json:"username"`
		Pass string `json:"password"`
	} `json:"irc"`
}

type ircConn struct {
	net.Conn
	scanner *bufio.Scanner
}

type ircMsg struct {
	src, cmd string
	args []string
}

func (irc ircConn) get() (m ircMsg, e error) {
	if !irc.scanner.Scan() {
		e = irc.scanner.Err()
		if e == nil {
			e = net.ErrClosed
		}
		return
	}
	l := irc.scanner.Text()
	c := 0
	if c = strings.IndexFunc(l, notSpace); c == -1 {
		return
	}
	l = l[c:]
	if strings.HasPrefix(l, ":") {
		l = l[1:]
		if c = strings.IndexFunc(l, unicode.IsSpace); c == -1 {
			return
		}
		if b := strings.IndexRune(l[:c], '!'); b != -1 {
			m.src = l[:b]
		} else {
			m.src = l[:c]
		}
		l = l[c:]
		if c = strings.IndexFunc(l, notSpace); c == -1 {
			return
		}
		l = l[c:]
	}
	if c = strings.IndexFunc(l, unicode.IsSpace); c == -1 {
		c = len(l)
	}
	m.cmd = l[:c]
	l = l[c:]
	for {
		if c = strings.IndexFunc(l, notSpace); c == -1 {
			break
		}
		l = l[c:]
		if strings.HasPrefix(l, ":") {
			m.args = append(m.args, l[1:])
			break
		}
		if c = strings.IndexFunc(l, unicode.IsSpace); c == -1 {
			c = len(l)
		}
		m.args = append(m.args, l[:c])
		l = l[c:]
	}
	return
}

func notSpace(r rune) bool {
	return !unicode.IsSpace(r)
}
