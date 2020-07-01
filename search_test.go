package sonic_test

import (
	"errors"
	"net"
	"testing"

	"github.com/stevecallear/sonic"
)

func TestNewSearch(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		exp     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			exp:     ErrConnect,
		},
		{
			name: "should return channel errors",
			setup: func(s *Server) {
				s.On(`^START search \w+$`).Send("ERR START")
			},
			exp: errors.New("START"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			tt.setup(server)

			server.Run(t, func(t *testing.T, conn net.Conn) {
				restore := SetDialTCP(func(string) (net.Conn, error) {
					return conn, tt.connErr
				})
				defer restore()

				search := sonic.NewSearch(sonic.Options{
					Password: "password",
				})
				defer search.Close()

				act := search.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}

func TestSearch_Query(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.QueryRequest
		exp     []string
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return pending errors",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^QUERY`).Send("ERR PENDING")
			},
			err: errors.New("PENDING"),
		},
		{
			name: "should return event errors",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^QUERY`).Send("PENDING z98uDE0f").Send("ERR EVENT")
			},
			err: errors.New("EVENT"),
		},
		{
			name: "should return query results",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^QUERY collection bucket \"term\"$`).
					Send("PENDING z98uDE0f").
					Send("EVENT QUERY z98uDE0f article:one article:two")
			},
			request: sonic.QueryRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Terms:      "term",
			},
			exp: []string{"article:one", "article:two"},
		},
		{
			name: "should use optional parameters",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^QUERY collection bucket \"term\" LIMIT\(5\) OFFSET\(10\) LANG\(eng\)$`).
					Send("PENDING z98uDE0f").
					Send("EVENT QUERY z98uDE0f article:one article:two")
			},
			request: sonic.QueryRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Terms:      "term",
				Limit:      5,
				Offset:     10,
				Lang:       "eng",
			},
			exp: []string{"article:one", "article:two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			tt.setup(server)

			server.Run(t, func(t *testing.T, conn net.Conn) {
				restore := SetDialTCP(func(string) (net.Conn, error) {
					return conn, tt.connErr
				})
				defer restore()

				search := sonic.NewSearch(sonic.Options{
					Password: "password",
				})
				defer search.Close()

				act, err := search.Query(tt.request)
				AssertError(t, err, tt.err)
				AssertDeepEqual(t, act, tt.exp)
			})
		})
	}
}

func TestSearch_Suggest(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		request sonic.SuggestRequest
		connErr error
		exp     []string
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return pending errors",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^SUGGEST`).Send("ERR PENDING")
			},
			err: errors.New("PENDING"),
		},
		{
			name: "should return event errors",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^SUGGEST`).Send("PENDING z98uDE0f").Send("ERR EVENT")
			},
			err: errors.New("EVENT"),
		},
		{
			name: "should return suggestions",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^SUGGEST collection bucket \"wor\"$`).
					Send("PENDING z98uDE0f").
					Send("EVENT SUGGEST z98uDE0f word worry")
			},
			request: sonic.SuggestRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Word:       "wor",
			},
			exp: []string{"word", "worry"},
		},
		{
			name: "should use optional parameters",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^SUGGEST collection bucket \"wor\" LIMIT\(5\)$`).
					Send("PENDING z98uDE0f").
					Send("EVENT SUGGEST z98uDE0f word worry")
			},
			request: sonic.SuggestRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Word:       "wor",
				Limit:      5,
			},
			exp: []string{"word", "worry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			tt.setup(server)

			server.Run(t, func(t *testing.T, conn net.Conn) {
				restore := SetDialTCP(func(string) (net.Conn, error) {
					return conn, tt.connErr
				})
				defer restore()

				search := sonic.NewSearch(sonic.Options{
					Password: "password",
				})
				defer search.Close()

				act, err := search.Suggest(tt.request)
				AssertError(t, err, tt.err)
				AssertDeepEqual(t, act, tt.exp)
			})
		})
	}
}

func TestSearch_Ping(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		exp     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			exp:     ErrConnect,
		},
		{
			name: "should return ping errors",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^PING$`).Send("ERR PING")
			},
			exp: errors.New("PING"),
		},
		{
			name: "should ping the server",
			setup: func(s *Server) {
				s.ConfigureStart("search", 20000)
				s.On(`^PING$`).Send("PONG")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			tt.setup(server)

			server.Run(t, func(t *testing.T, conn net.Conn) {
				restore := SetDialTCP(func(string) (net.Conn, error) {
					return conn, tt.connErr
				})
				defer restore()

				search := sonic.NewSearch(sonic.Options{
					Password: "password",
				})
				defer search.Close()

				act := search.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}
