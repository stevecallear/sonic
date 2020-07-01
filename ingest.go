package sonic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stevecallear/sonic/pool"
)

type (
	// Ingest represents an ingest client
	Ingest struct {
		*client
	}

	// PushRequest represents a PUSH request
	PushRequest struct {
		Collection string
		Bucket     string
		Object     string
		Text       string
		Lang       string // optional
	}

	// PopRequest represents a POP request
	PopRequest struct {
		Collection string
		Bucket     string
		Object     string
		Text       string
	}

	//CountRequest represents a COUNT request
	CountRequest struct {
		Collection string
		Bucket     string // optional
		Object     string // optional
	}

	// FlushRequest represents a FLUSH request
	FlushRequest struct {
		Collection string
		Bucket     string // optional
		Object     string // optional
	}
)

// NewIngest returns a new ingest client
func NewIngest(o Options) *Ingest {
	return &Ingest{
		client: newClient("ingest", o),
	}
}

// Push pushes search data to the index
func (i *Ingest) Push(r PushRequest) error {
	return i.pool.Exec(func(c pool.Channel) error {
		for _, t := range c.Split(r.Text) {
			msg := fmt.Sprintf("PUSH %s %s %s \"%s\"", r.Collection, r.Bucket, r.Object, t)
			msg = appendLang(msg, r.Lang)

			err := c.Write(msg)
			if err != nil {
				return err
			}

			// OK
			_, err = c.Read()
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// Pop pops search data from the index
func (i *Ingest) Pop(r PopRequest) (int, error) {
	res, err := i.pool.Query(func(c pool.Channel) (interface{}, error) {
		var nt int
		for _, t := range c.Split(r.Text) {
			err := c.Write(fmt.Sprintf("POP %s %s %s \"%s\"", r.Collection, r.Bucket, r.Object, t))
			if err != nil {
				return nt, err
			}

			// RESULT <n>
			res, err := c.Read()
			if err != nil {
				return nt, err
			}

			n, err := strconv.Atoi(strings.Split(res, " ")[1])
			if err != nil {
				return nt, err
			}

			nt += n
		}

		return nt, nil
	})
	if err != nil {
		return 0, err
	}

	return res.(int), nil
}

// Count counts indexed search data
func (i *Ingest) Count(r CountRequest) (int, error) {
	res, err := i.pool.Query(func(c pool.Channel) (interface{}, error) {
		var msg string
		switch {
		case r.Bucket != "" && r.Object != "":
			msg = fmt.Sprintf("COUNT %s %s %s", r.Collection, r.Bucket, r.Object)
		case r.Bucket != "":
			msg = fmt.Sprintf("COUNT %s %s", r.Collection, r.Bucket)
		default:
			msg = fmt.Sprintf("COUNT %s", r.Collection)
		}

		err := c.Write(msg)
		if err != nil {
			return nil, err
		}

		// RESULT <count>
		res, err := c.Read()
		if err != nil {
			return nil, err
		}

		return strconv.Atoi(strings.Split(res, " ")[1])
	})
	if err != nil {
		return 0, err
	}

	return res.(int), nil
}

// Flush flushes all indexed data from a collection, bucket or object
func (i *Ingest) Flush(r FlushRequest) (int, error) {
	res, err := i.pool.Query(func(c pool.Channel) (interface{}, error) {
		var msg string
		switch {
		case r.Bucket != "" && r.Object != "":
			msg = fmt.Sprintf("FLUSHO %s %s %s", r.Collection, r.Bucket, r.Object)
		case r.Bucket != "":
			msg = fmt.Sprintf("FLUSHB %s %s", r.Collection, r.Bucket)
		default:
			msg = fmt.Sprintf("FLUSHC %s", r.Collection)
		}

		err := c.Write(msg)
		if err != nil {
			return nil, err
		}

		// RESULT <count>
		res, err := c.Read()
		if err != nil {
			return nil, err
		}

		return strconv.Atoi(strings.Split(res, " ")[1])
	})
	if err != nil {
		return 0, err
	}

	return res.(int), nil
}
