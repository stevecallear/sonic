package pool

import (
	"errors"
	"io"
	"sync"
	"time"
)

type (
	// Pool represents a pool
	Pool struct {
		newFn   func() (Channel, error)
		items   chan Channel
		curSize int
		maxSize int
		timeout time.Duration
		mu      *sync.Mutex
	}

	// Options represents a set of pool options
	Options struct {
		NewFn   func() (Channel, error)
		Size    int
		Timeout time.Duration
	}

	// Channel represents a sonic channel
	Channel interface {
		Write(string) error
		Read() (string, error)
		Split(string) []string
		Close() error
	}
)

// ErrTimeout indicates that a timeout occurred waiting for an available item
var ErrTimeout = errors.New("pool: timeout waiting for available item")

// New returns a new pool for specified options
func New(o Options) *Pool {
	if o.Size <= 0 {
		o.Size = 1
	}
	if o.Timeout <= 0 {
		o.Timeout = 30 * time.Second
	}

	return &Pool{
		newFn:   o.NewFn,
		items:   make(chan Channel, o.Size),
		maxSize: o.Size,
		timeout: o.Timeout,
		mu:      new(sync.Mutex),
	}
}

// Exec executes against the next available channel
func (p *Pool) Exec(fn func(Channel) error) error {
	c, err := p.next()
	if err != nil {
		return err
	}

	err = fn(c)
	if err == io.EOF {
		p.remove(c)
		return err
	}

	p.restore(c)
	return err
}

// Query queries the next available channel
func (p *Pool) Query(fn func(Channel) (interface{}, error)) (interface{}, error) {
	c, err := p.next()
	if err != nil {
		return nil, err
	}

	res, err := fn(c)
	if err == io.EOF {
		p.remove(c)
		return res, err
	}

	p.restore(c)
	return res, err
}

// Close closes all pool channels
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.items)
	for c := range p.items {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pool) new() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.curSize >= p.maxSize {
		return nil
	}

	c, err := p.newFn()
	if err != nil {
		return err
	}

	p.items <- c
	p.curSize++

	return nil
}

func (p *Pool) next() (Channel, error) {
	if len(p.items) < 1 {
		if err := p.new(); err != nil {
			return nil, err
		}
	}

	select {
	case c := <-p.items:
		return c, nil
	case <-time.After(p.timeout):
		return nil, ErrTimeout
	}
}

func (p *Pool) restore(c Channel) {
	p.items <- c
}

func (p *Pool) remove(c Channel) {
	p.mu.Lock()
	defer p.mu.Unlock()

	c.Close()
	p.curSize--
}
