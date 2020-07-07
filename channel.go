package sonic

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type channel struct {
	conn     net.Conn
	reader   *bufio.Reader
	logFn    func(string)
	maxRunes int
}

var (
	// DialTCP connects to the specified server
	DialTCP = func(addr string) (net.Conn, error) {
		return net.Dial("tcp", addr)
	}

	// ErrInvalidResponse indicates that the received response message is invalid
	ErrInvalidResponse = errors.New("invalid response")

	bufferRegex = regexp.MustCompile(`^.+buffer\(([0-9]+)\)$`)
)

func newChannel(ctype string, o Options) (*channel, error) {
	conn, err := DialTCP(o.Addr)
	if err != nil {
		return nil, err
	}

	close := func(err error) error {
		conn.Close()
		return err
	}

	c := &channel{
		conn:   conn,
		reader: bufio.NewReader(conn),
		logFn:  o.LogFn,
	}
	if c.logFn == nil {
		c.logFn = func(string) {}
	}

	err = c.Write(fmt.Sprintf("START %s %s", ctype, o.Password))
	if err != nil {
		return nil, close(err)
	}

	_, err = c.Read()
	if err != nil {
		return nil, close(err)
	}

	res, err := c.Read()
	if err != nil {
		return nil, close(err)
	}

	mr, err := parseMaxRunes(res)
	if err != nil {
		return nil, close(err)
	}

	c.maxRunes = mr
	return c, nil
}

func (c *channel) Read() (string, error) {
	s, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(s, "ERR ") {
		return "", errors.New(strings.TrimSpace(s[4:]))
	}

	s = strings.TrimSpace(s)
	c.logFn(s)
	return s, nil
}

func (c *channel) Write(s string) error {
	c.logFn(s)
	_, err := c.conn.Write([]byte(s + "\r\n"))
	return err
}

func (c *channel) Close() error {
	err := c.Write("QUIT")
	if err != nil {
		return err
	}

	_, err = c.Read()
	if err != nil {
		return err
	}

	return c.conn.Close()
}

func (c *channel) Split(s string) []string {
	ss := []string{}
	rs := []rune(s)

	for i := 0; i < len(rs); i += c.maxRunes {
		nn := i + c.maxRunes
		if nn > len(rs) {
			nn = len(rs)
		}
		ss = append(ss, string(rs[i:nn]))
	}

	return ss
}

func (c *channel) Escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")

	return s
}

func parseMaxRunes(msg string) (int, error) {
	m := bufferRegex.FindStringSubmatch(msg)
	if len(m) != 2 {
		return 0, ErrInvalidResponse
	}

	b, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, err
	}

	// allow half of the buffer for text runes at 4 bytes each
	return b / 2 / 4, nil
}
