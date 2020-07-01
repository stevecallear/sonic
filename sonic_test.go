package sonic_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stevecallear/sonic"
)

type (
	Server struct {
		client    net.Conn
		conn      net.Conn
		reader    *bufio.Reader
		responses []*Response
	}

	Response struct {
		regex   *regexp.Regexp
		data    []string
		matched bool
	}
)

var ErrConnect = errors.New("CONNECT")

func NewServer() *Server {
	c, s := net.Pipe()

	return &Server{
		client:    c,
		conn:      s,
		reader:    bufio.NewReader(s),
		responses: []*Response{},
	}
}

func (s *Server) ConfigureStart(ctype string, maxBufferBytes int) *Server {
	s.On(fmt.Sprintf("^START %s \\w+$", ctype)).
		Send("CONNECTED <sonic-server v1.2.3>").
		Send(fmt.Sprintf("STARTED search protocol(1) buffer(%d)", maxBufferBytes))

	return s
}

func (s *Server) On(pattern string) *Response {
	r := &Response{
		regex: regexp.MustCompile(pattern),
		data:  []string{},
	}

	s.responses = append(s.responses, r)
	return r
}

func (s *Server) Run(t *testing.T, fn func(*testing.T, net.Conn)) {
	go func() {
		for {
			str, err := s.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				panic(err)
			}

			str = strings.TrimSpace(str)
			if strings.HasPrefix(str, "QUIT") {
				s.conn.Write([]byte("ENDED quit\r\n"))
				return
			}

			var ok bool
			var msgs []string
			for _, r := range s.responses {
				if msgs, ok = r.match(str); ok {
					for _, msg := range msgs {
						_, err = s.conn.Write([]byte(msg + "\r\n"))
						if err != nil {
							panic(err)
						}
					}
					break
				}
			}

			if !ok {
				_, err = s.conn.Write([]byte(fmt.Sprintf("ERR no match: %s \r\n", str)))
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	fn(t, s.client)

	for _, r := range s.responses {
		if !r.matched {
			t.Errorf("not matched: \"%s\"", r.regex)
		}
	}
}

func (s *Server) Close() error {
	return s.conn.Close()
}

func (r *Response) Send(data string) *Response {
	r.data = append(r.data, data)
	return r
}

func (r *Response) match(msg string) ([]string, bool) {
	if r.regex.MatchString(msg) {
		r.matched = true
		return r.data, true
	}
	return nil, false
}

func SetDialTCP(fn func(string) (net.Conn, error)) func() {
	pfn := sonic.DialTCP
	sonic.DialTCP = fn

	return func() {
		sonic.DialTCP = pfn
	}
}

func AssertError(t *testing.T, act, exp error) {
	if act == exp {
		return
	}

	if act != nil && exp != nil {
		if act.Error() == exp.Error() {
			return
		}
	}

	t.Errorf("got %v, expected %v", act, exp)
}

func AssertEqual(t *testing.T, act, exp interface{}) {
	if act != exp {
		t.Errorf("got %d, expected %d", act, exp)
	}
}

func AssertDeepEqual(t *testing.T, act, exp interface{}) {
	if !reflect.DeepEqual(act, exp) {
		t.Errorf("got %v, expected %v", act, exp)
	}
}
