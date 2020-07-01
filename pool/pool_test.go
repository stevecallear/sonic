package pool_test

//go:generate mockgen -source=pool.go -destination=mocks/pool.go -package=mocks

import (
	"errors"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stevecallear/sonic/pool"
	"github.com/stevecallear/sonic/pool/mocks"
)

func TestPool_Exec(t *testing.T) {
	err := errors.New("error")

	tests := []struct {
		name   string
		setup  func(*mocks.MockChannelMockRecorder)
		newErr error
		exec   func(pool.Channel) error
		err    error
	}{
		{
			name:   "should return create errors",
			setup:  func(r *mocks.MockChannelMockRecorder) {},
			newErr: err,
			err:    err,
		},
		{
			name:  "should return func errors",
			setup: func(r *mocks.MockChannelMockRecorder) {},
			exec: func(pool.Channel) error {
				return err
			},
			err: err,
		},
		{
			name: "should remove broken channels",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Close().Return(nil).Times(1)
			},
			exec: func(pool.Channel) error {
				return io.EOF
			},
			err: io.EOF,
		},
		{
			name: "should execute channel actions",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Write("req").Return(nil).Times(1)
				r.Read().Return("res", nil).Times(1)
			},
			exec: func(c pool.Channel) error {
				c.Write("req")
				_, err := c.Read()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			p := pool.New(pool.Options{
				NewFn: func() (pool.Channel, error) {
					c := mocks.NewMockChannel(ctrl)
					tt.setup(c.EXPECT())

					return c, tt.newErr
				},
			})

			err := p.Exec(tt.exec)
			if err != tt.err {
				t.Errorf("got %v, expected %v", err, tt.err)
			}
		})
	}
}

func TestPool_Query(t *testing.T) {
	err := errors.New("error")

	tests := []struct {
		name   string
		setup  func(*mocks.MockChannelMockRecorder)
		newErr error
		query  func(pool.Channel) (interface{}, error)
		exp    interface{}
		err    error
	}{
		{
			name:   "should return create errors",
			setup:  func(r *mocks.MockChannelMockRecorder) {},
			newErr: err,
			err:    err,
		},
		{
			name:  "should return query errors",
			setup: func(r *mocks.MockChannelMockRecorder) {},
			query: func(pool.Channel) (interface{}, error) {
				return nil, err
			},
			err: err,
		},
		{
			name: "should remove broken channels",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Close().Return(nil).Times(1)
			},
			query: func(pool.Channel) (interface{}, error) {
				return nil, io.EOF
			},
			err: io.EOF,
		},
		{
			name: "should execute query operations",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Write("req").Return(nil).Times(1)
				r.Read().Return("res", nil).Times(1)
			},
			query: func(c pool.Channel) (interface{}, error) {
				c.Write("req")
				return c.Read()
			},
			exp: "res",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			p := pool.New(pool.Options{
				NewFn: func() (pool.Channel, error) {
					c := mocks.NewMockChannel(ctrl)
					tt.setup(c.EXPECT())
					return c, tt.newErr
				},
			})

			act, err := p.Query(tt.query)
			if err != tt.err {
				t.Errorf("got %v, expected %v", err, tt.err)
			}

			if act != tt.exp {
				t.Errorf("got %s, expected %v", act, tt.exp)
			}
		})
	}
}

func TestPool_Close(t *testing.T) {
	err := errors.New("error")

	tests := []struct {
		name  string
		setup func(*mocks.MockChannelMockRecorder)
		err   error
	}{
		{
			name: "should return close errors",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Close().Return(err).Times(1)
			},
			err: err,
		},
		{
			name: "should close all channels",
			setup: func(r *mocks.MockChannelMockRecorder) {
				r.Close().Return(nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			p := pool.New(pool.Options{
				NewFn: func() (pool.Channel, error) {
					c := mocks.NewMockChannel(ctrl)
					tt.setup(c.EXPECT())
					return c, nil
				},
			})

			// force a channel to be created
			p.Exec(func(pool.Channel) error {
				return nil
			})

			err := p.Close()
			if err != tt.err {
				t.Errorf("got %v, expected %v", err, tt.err)
			}
		})
	}
}
