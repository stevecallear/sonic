package sonic

import (
	"fmt"
	"strings"

	"github.com/stevecallear/sonic/pool"
)

type (
	// Search represents a search client
	Search struct {
		*client
	}

	// QueryRequest represents a query request
	QueryRequest struct {
		Collection string
		Bucket     string
		Terms      string
		Limit      int    // optional
		Offset     int    // optional
		Lang       string // optional
	}

	// SuggestRequest represents a suggest request
	SuggestRequest struct {
		Collection string
		Bucket     string
		Word       string
		Limit      int // optional
	}
)

// NewSearch returns a new search client
func NewSearch(o Options) *Search {
	return &Search{
		client: newClient("search", o),
	}
}

// Query returns a list of objects matching the specified query
func (s *Search) Query(r QueryRequest) ([]string, error) {
	res, err := s.pool.Query(func(c pool.Channel) (interface{}, error) {
		msg := fmt.Sprintf("QUERY %s %s \"%s\"", r.Collection, r.Bucket, r.Terms)
		msg = appendLang(appendOffset(appendLimit(msg, r.Limit), r.Offset), r.Lang)

		err := c.Write(msg)
		if err != nil {
			return nil, err
		}

		// PENDING [marker]
		_, err = c.Read()
		if err != nil {
			return nil, err
		}

		// EVENT QUERY [marker] [o1] [o2]
		return c.Read()
	})
	if err != nil {
		return nil, err
	}

	return strings.Split(res.(string), " ")[3:], nil
}

// Suggest returns a list of word suggestions based on the specified input
func (s *Search) Suggest(r SuggestRequest) ([]string, error) {
	res, err := s.pool.Query(func(c pool.Channel) (interface{}, error) {
		msg := fmt.Sprintf("SUGGEST %s %s \"%s\"", r.Collection, r.Bucket, r.Word)
		msg = appendLimit(msg, r.Limit)

		err := c.Write(msg)
		if err != nil {
			return "", err
		}

		// PENDING [marker]
		_, err = c.Read()
		if err != nil {
			return "", err
		}

		// EVENT SUGGEST [marker] [t1] [t2] ...
		return c.Read()
	})
	if err != nil {
		return nil, err
	}

	return strings.Split(res.(string), " ")[3:], nil
}

func appendLimit(msg string, limit int) string {
	if limit > 0 {
		msg = fmt.Sprintf("%s LIMIT(%d)", msg, limit)
	}
	return msg
}

func appendOffset(msg string, offset int) string {
	if offset > 0 {
		msg = fmt.Sprintf("%s OFFSET(%d)", msg, offset)
	}
	return msg
}

func appendLang(msg, lang string) string {
	if lang != "" {
		msg = fmt.Sprintf("%s LANG(%s)", msg, lang)
	}
	return msg
}
