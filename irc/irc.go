package irc

import (
	"io"
	"fmt"
	"net"
	"bufio"
	"unicode"
	"strings"
)

type Connection struct {
	conn net.Conn
	scanner *bufio.Scanner
}

func (c Connection) Read() (Message, error) {
	msg := Message{}

	for {
		if !c.scanner.Scan() {
			e := c.scanner.Err()
			if e == nil {
				e = io.EOF
			}
			return Message{}, e
		}

		line := c.scanner.Text()

		r, ok := findNextField(line, 0)
		if ok && line[r.Start] == ':' {
			bang := strings.Index(line[r.Start:r.Start + r.Length], "!")
			if bang != -1 {
				msg.Source = line[r.Start + 1:bang]
			} else {
				msg.Source = line[r.Start + 1:r.Start + r.Length]
			}
			r, ok = findNextField(line, r.Start + r.Length + 1)
		}
		if ok {
			msg.Command = line[r.Start:r.Start + r.Length]
			r, ok = findNextField(line, r.Start + r.Length + 1)
		} else {
			continue
		}
		for ok && line[r.Start] != ':' {
			msg.Params = append(msg.Params, line[r.Start:r.Start + r.Length])
			r, ok = findNextField(line, r.Start + r.Length + 1)
		}
		if ok {
			msg.Params = append(
				msg.Params,
				line[r.Start + 1:strings.LastIndexFunc(line, isNotSpace) + 1],
			)
		}

		break
	}

	return msg, nil
}

func (c Connection) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c Connection) Close() {
	c.conn.Close()
}

type Message struct {
	Source string
	Command string
	Params []string
}

func Connect(host string, port int, username string, password string) (Connection, error) {
	conn, e := net.Dial("tcp", fmt.Sprintf("%v:%v", host, port))
	if e != nil {
		return Connection{}, e
	}

	c := Connection{ conn: conn, scanner: bufio.NewScanner(conn) }

	if _, e := fmt.Fprintf(c, "PASS %v\nNICK %v\n", password, username); e != nil {
		c.Close()
		return Connection{}, e
	}

	return c, nil
}

type fieldRange struct {
	Start int
	Length int
}

func isNotSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

func findNextField(str string, start int) (fieldRange, bool) {
	r := fieldRange{}

	if start >= len(str) {
		return r, false
	}

	r.Start = strings.IndexFunc(str[start:], isNotSpace)
	if r.Start == -1 {
		return r, false
	}

	r.Start += start

	r.Length = strings.IndexFunc(str[r.Start:], unicode.IsSpace)
	if r.Length == -1 {
		r.Length = len(str) - r.Start
	}

	return r, true
}
