package sonic

import (
	"time"

	"github.com/stevecallear/sonic/pool"
)

type (
	// Options represents a set of client options
	Options struct {
		Addr        string
		Password    string
		PoolSize    int
		PoolTimeout time.Duration
		LogFn       func(string)
	}

	client struct {
		pool *pool.Pool
	}
)

func newClient(ctype string, o Options) *client {
	return &client{
		pool: pool.New(pool.Options{
			NewFn: func() (pool.Channel, error) {
				return newChannel(ctype, o)
			},
			Size:    o.PoolSize,
			Timeout: o.PoolTimeout,
		}),
	}
}

func (c *client) Ping() error {
	return c.pool.Exec(func(ch pool.Channel) error {
		err := ch.Write("PING")
		if err != nil {
			return err
		}

		_, err = ch.Read()
		return err
	})
}

func (c *client) Close() error {
	return c.pool.Close()
}
