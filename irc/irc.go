package irc

import (
	"io"
	"fmt"
	"net"
	"time"
	"bufio"
	"unicode"
	"strings"
	"strconv"
	"osubot/util"
)

var RateLimit float32 = 4

type Message struct {
	Source string
	Command string
	Params []string
}

type Connection struct {
	user string
	conn net.Conn
	scanner *bufio.Scanner
	requests chan []byte
}

func Connect(host string, port int, username string, password string) (Connection, error) {
	conn, e := net.Dial("tcp", fmt.Sprintf("%v:%v", host, port))
	if e != nil {
		return Connection{}, e
	}

	c := Connection{ 
		user: username, 
		conn: conn, 
		scanner: bufio.NewScanner(conn),
		requests: make(chan []byte, 64),
	}

	go func(){
		for b := range c.requests {
			util.IrcLogger.Println("Sent", strconv.Quote(string(b)))
			c.conn.Write(b)
			time.Sleep(time.Duration(1 / RateLimit) * time.Second)
		}
	}()

	fmt.Fprintf(c.conn, "PASS %v\nNICK %v\n", password, username)
	return c, nil
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

	if msg.Command != "QUIT" {
		util.IrcLogger.Printf("%v: %v %v", msg.Source, msg.Command, strings.Join(msg.Params, " "))
	}

	return msg, nil
}

func (c Connection) Write(b []byte) (int, error) {
	b2 := make([]byte, len(b))
	copy(b2, b)
	c.requests <- b2
	return len(b), nil
}

func (c Connection) Close() {
	c.conn.Close()
	close(c.requests)
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
