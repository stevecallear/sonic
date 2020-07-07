package sonic_test

import (
	"errors"
	"net"
	"testing"

	"github.com/stevecallear/sonic"
)

func TestNewIngest(t *testing.T) {
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
				s.On(`^START ingest \w+$`).Send("ERR START")
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				act := ingest.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}

func TestIngest_Push(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.PushRequest
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return push errors",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^PUSH").Send("ERR PUSH")
			},
			request: sonic.PushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
			},
			err: errors.New("PUSH"),
		},
		{
			name: "should push the text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^PUSH collection bucket object "text"$`).Send("OK")
			},
			request: sonic.PushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
			},
		},
		{
			name: "should use optional parameters",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^PUSH collection bucket object "text" LANG\(eng\)$`).Send("OK")
			},
			request: sonic.PushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
				Lang:       "eng",
			},
		},
		{
			name: "should split long text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 40) // 5 runes * 4 bytes * 2 = 40
				s.On(`^PUSH collection bucket object "long "$`).Send("OK")
				s.On(`^PUSH collection bucket object "text"$`).Send("OK")
			},
			request: sonic.PushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "long text",
			},
		},
		{
			name: "should escape text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^PUSH collection bucket object "\\\\ \\n \\\""$`).Send("OK")
			},
			request: sonic.PushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "\\ \n \"",
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				err := ingest.Push(tt.request)
				AssertError(t, err, tt.err)
			})
		})
	}
}

func TestIngest_Pop(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.PopRequest
		exp     int
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return pop errors",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^POP`).Send("ERR POP")
			},
			request: sonic.PopRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
			},
			err: errors.New("POP"),
		},
		{
			name: "should return an error if the result cannot be parsed",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^POP").Send("RESULT invalid")
			},
			request: sonic.PopRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
			},
			err: sonic.ErrInvalidResponse,
		},
		{
			name: "should pop the text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^POP collection bucket object "text"$`).Send("RESULT 10")
			},
			request: sonic.PopRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "text",
			},
			exp: 10,
		},
		{
			name: "should split long text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 40) // 5 runes * 4 bytes * 2 = 40
				s.On(`^POP collection bucket object "long "$`).Send("RESULT 3")
				s.On(`^POP collection bucket object "text"$`).Send("RESULT 7")
			},
			request: sonic.PopRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "long text",
			},
			exp: 10,
		},
		{
			name: "should escape text",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On(`^POP collection bucket object "\\\\ \\n \\\""$`).Send("RESULT 10")
			},
			request: sonic.PopRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
				Text:       "\\ \n \"",
			},
			exp: 10,
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				act, err := ingest.Pop(tt.request)
				AssertError(t, err, tt.err)
				AssertEqual(t, act, tt.exp)
			})
		})
	}
}

func TestIngest_Count(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.CountRequest
		exp     int
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return count errors",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^COUNT").Send("ERR COUNT")
			},
			request: sonic.CountRequest{
				Collection: "collection",
			},
			err: errors.New("COUNT"),
		},
		{
			name: "should return an error if the result cannot be parsed",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^COUNT").Send("RESULT invalid")
			},
			request: sonic.CountRequest{
				Collection: "collection",
			},
			err: sonic.ErrInvalidResponse,
		},
		{
			name: "should count the collection",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^COUNT collection$").Send("RESULT 10")
			},
			request: sonic.CountRequest{
				Collection: "collection",
			},
			exp: 10,
		},
		{
			name: "should count the bucket",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^COUNT collection bucket$").Send("RESULT 10")
			},
			request: sonic.CountRequest{
				Collection: "collection",
				Bucket:     "bucket",
			},
			exp: 10,
		},
		{
			name: "should count the object",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^COUNT collection bucket object$").Send("RESULT 10")
			},
			request: sonic.CountRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
			},
			exp: 10,
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				act, err := ingest.Count(tt.request)
				AssertError(t, err, tt.err)
				AssertEqual(t, act, tt.exp)
			})
		})
	}
}

func TestIngest_Flush(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.FlushRequest
		exp     int
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return flush errors",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^FLUSHC").Send("ERR FLUSHC")
			},
			request: sonic.FlushRequest{
				Collection: "collection",
			},
			err: errors.New("FLUSHC"),
		},
		{
			name: "should flush the collection",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^FLUSHC collection$").Send("RESULT 10")
			},
			request: sonic.FlushRequest{
				Collection: "collection",
			},
			exp: 10,
		},
		{
			name: "should flush the bucket",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^FLUSHB collection bucket$").Send("RESULT 10")
			},
			request: sonic.FlushRequest{
				Collection: "collection",
				Bucket:     "bucket",
			},
			exp: 10,
		},
		{
			name: "should flush the object",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^FLUSHO collection bucket object$").Send("RESULT 10")
			},
			request: sonic.FlushRequest{
				Collection: "collection",
				Bucket:     "bucket",
				Object:     "object",
			},
			exp: 10,
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				act, err := ingest.Flush(tt.request)
				AssertError(t, err, tt.err)
				AssertEqual(t, act, tt.exp)
			})
		})
	}
}

func TestIngest_Ping(t *testing.T) {
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
				s.ConfigureStart("ingest", 20000)
				s.On("^PING$").Send("ERR PING")
			},
			exp: errors.New("PING"),
		},
		{
			name: "should ping the server",
			setup: func(s *Server) {
				s.ConfigureStart("ingest", 20000)
				s.On("^PING$").Send("PONG")
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

				ingest := sonic.NewIngest(sonic.Options{
					Password: "password",
				})
				defer ingest.Close()

				act := ingest.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}
