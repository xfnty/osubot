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
	conn net.Conn
	scanner *bufio.Scanner
	limiter *time.Ticker
}

type Msg struct {
	Src, Cmd string
	Args []string
}

func Connect(addr string, rateLimit float32) (c Conn, e error) {
	c.conn, e = net.Dial("tcp", addr)
	if e != nil {
		return
	}
	c.scanner = bufio.NewScanner(c.conn)
	c.limiter = time.NewTicker(time.Duration((1 / rateLimit) * float32(time.Second)))
	return
}

func (c Conn) Close() error {
	return c.conn.Close()
}

func (c Conn) Send(cmd string, args ...any) {
	<-c.limiter.C

	sArgs := make([]string, len(args), len(args))
	for i, a := range args {
		sArgs[i] = fmt.Sprintf("%v", a)
	}

	c.conn.Write([]byte(fmt.Sprintf("%v %v\n", cmd, strings.Join(sArgs, " "))))
}

func (conn Conn) Recv() (m Msg, e error) {
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
