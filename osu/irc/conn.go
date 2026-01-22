package irc

import (
	"fmt"
	"net"
	"time"
	"bufio"
	"strings"
	"unicode"
)

type Conn struct {
	net.Conn
	scanner *bufio.Scanner
	sendDelay time.Duration
	sendQueue chan []byte
}

type Msg struct {
	Src, Cmd string
	Args []string
}

func Connect(addr string, rateLimit float32) (c Conn, e error) {
	c.Conn, e = net.Dial("tcp", addr)
	if e != nil {
		return
	}
	c.scanner = bufio.NewScanner(c.Conn)
	c.sendDelay = time.Duration((1 / rateLimit) * float32(time.Second))
	c.sendQueue = make(chan []byte)

	go func(){
		for b := range c.sendQueue {
			c.Conn.Write(b)
			time.Sleep(c.sendDelay)
		}
	}()

	return
}

func (c Conn) Write(b []byte) (int, error) {
	c.sendQueue <- b
	return len(b), nil
}

func (c Conn) Close() error {
	close(c.sendQueue)
	return c.Conn.Close()
}

func (conn Conn) Get() (m Msg, e error) {
	if !conn.scanner.Scan() {
		e = conn.scanner.Err()
		if e == nil {
			e = net.ErrClosed
		}
		return
	}
	l := conn.scanner.Text()
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
			m.Src = l[:b]
		} else {
			m.Src = l[:c]
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
	m.Cmd = l[:c]
	l = l[c:]
	for {
		if c = strings.IndexFunc(l, notSpace); c == -1 {
			break
		}
		l = l[c:]
		if strings.HasPrefix(l, ":") {
			m.Args = append(m.Args, l[1:])
			break
		}
		if c = strings.IndexFunc(l, unicode.IsSpace); c == -1 {
			c = len(l)
		}
		m.Args = append(m.Args, l[:c])
		l = l[c:]
	}
	return
}

func notSpace(r rune) bool {
	return !unicode.IsSpace(r)
}
