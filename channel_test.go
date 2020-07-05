package sonic_test

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stevecallear/sonic"
)

func TestNewChannel(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Server)
		err   error
	}{
		{
			name: "should return an error if the connected response is error",
			setup: func(s *Server) {
				s.On(`^START control \w+$`).Send("ERR START")
			},
			err: errors.New("START"),
		},
		{
			name: "should return an error if the started response is error",
			setup: func(s *Server) {
				s.On(`^START control \w+$`).
					Send("CONNECTED <sonic-server v1.2.3>").
					Send("ERR START")
			},
			err: errors.New("START"),
		},
		{
			name: "should return an error if the started response is invalid",
			setup: func(s *Server) {
				s.On(`^START control \w+$`).
					Send("CONNECTED <sonic-server v1.2.3>").
					Send(fmt.Sprintf("STARTED invalid"))
			},
			err: sonic.ErrInvalidResponse,
		},
		{
			name: "should return an error if the max buffer cannot be parsed",
			setup: func(s *Server) {
				s.On(`^START control \w+$`).
					Send("CONNECTED <sonic-server v1.2.3>").
					Send(fmt.Sprintf("STARTED search protocol(1) buffer(invalid)"))
			},
			err: sonic.ErrInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer()
			tt.setup(s)

			s.Run(t, func(t *testing.T, conn net.Conn) {
				restore := SetDialTCP(func(string) (net.Conn, error) {
					return conn, nil
				})
				defer restore()

				c := sonic.NewControl(sonic.Options{
					Password: "password",
				})

				err := c.Ping()
				AssertError(t, err, tt.err)
			})
		})
	}
}
