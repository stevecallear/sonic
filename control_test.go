package sonic_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stevecallear/sonic"
)

func TestNewControl(t *testing.T) {
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
				s.On(`^START control \w+$`).Send("ERR START")
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

				control := sonic.NewControl(sonic.Options{
					Password: "password",
				})
				defer control.Close()

				act := control.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}

func TestControl_Trigger(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		request sonic.TriggerRequest
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return trigger errors",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^TRIGGER").Send("ERR TRIGGER")
			},
			err: errors.New("TRIGGER"),
		},
		{
			name: "should trigger the action",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^TRIGGER action$").Send("OK")
			},
			request: sonic.TriggerRequest{
				Action: "action",
			},
		},
		{
			name: "should include optional parameters",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^TRIGGER action data$").Send("OK")
			},
			request: sonic.TriggerRequest{
				Action: "action",
				Data:   "data",
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

				control := sonic.NewControl(sonic.Options{
					Password: "password",
				})
				defer control.Close()

				err := control.Trigger(tt.request)
				AssertError(t, err, tt.err)
			})
		})
	}
}

func TestControl_Info(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Server)
		connErr error
		exp     sonic.InfoResponse
		err     error
	}{
		{
			name:    "should return connect errors",
			setup:   func(*Server) {},
			connErr: ErrConnect,
			err:     ErrConnect,
		},
		{
			name: "should return info errors",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^INFO$").Send("ERR INFO")
			},
			err: errors.New("INFO"),
		},
		{
			name: "should return an error if the response is invalid",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^INFO$").Send("INVALID")
			},
			err: sonic.ErrInvalidResponse,
		},
		{
			name: "should return an error if the response cannot be parsed",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^INFO$").Send("RESULT uptime(a) clients_connected(b) commands_total(c) command_latency_best(d) command_latency_worst(e) kv_open_count(f) fst_open_count(g) fst_consolidate_count(h)")
			},
			err: sonic.ErrInvalidResponse,
		},
		{
			name: "should return the server info",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
				s.On("^INFO$").Send("RESULT uptime(18) clients_connected(2) commands_total(1) command_latency_best(3) command_latency_worst(4) kv_open_count(5) fst_open_count(6) fst_consolidate_count(7)")
			},
			exp: sonic.InfoResponse{
				Uptime:              18 * time.Second,
				ClientsConnected:    2,
				CommandsTotal:       1,
				CommandLatencyBest:  3 * time.Millisecond,
				CommandLatencyWorst: 4 * time.Millisecond,
				KVOpenCount:         5,
				FSTOpenCount:        6,
				FSTConsolidateCount: 7,
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

				control := sonic.NewControl(sonic.Options{
					Password: "password",
				})
				defer control.Close()

				act, err := control.Info()
				AssertError(t, err, tt.err)
				AssertEqual(t, act, tt.exp)
			})
		})
	}
}

func TestControl_Ping(t *testing.T) {
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
				s.ConfigureStart("control", 20000)
				s.On("^PING$").Send("ERR PING")
			},
			exp: errors.New("PING"),
		},
		{
			name: "should ping the server",
			setup: func(s *Server) {
				s.ConfigureStart("control", 20000)
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

				control := sonic.NewControl(sonic.Options{
					Password: "password",
				})
				defer control.Close()

				act := control.Ping()
				AssertError(t, act, tt.exp)
			})
		})
	}
}
