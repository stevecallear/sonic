package sonic

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/stevecallear/sonic/pool"
)

type (
	// Control represents a control client
	Control struct {
		*client
	}

	// TriggerRequest represents a trigger request
	TriggerRequest struct {
		Action string
		Data   string // optional
	}

	// InfoResponse represents an info response
	InfoResponse struct {
		Uptime              time.Duration
		ClientsConnected    int
		CommandsTotal       int
		CommandLatencyBest  time.Duration
		CommandLatencyWorst time.Duration
		KVOpenCount         int
		FSTOpenCount        int
		FSTConsolidateCount int
	}
)

var (
	// ErrInvalidInfoResponse indicates the the response from INFO is invalid
	ErrInvalidInfoResponse = errors.New("control: invalid INFO response")

	infoRegexp = regexp.MustCompile(`^RESULT uptime\((\d+)\) clients_connected\((\d+)\) commands_total\((\d+)\) command_latency_best\((\d+)\) command_latency_worst\((\d+)\) kv_open_count\((\d+)\) fst_open_count\((\d+)\) fst_consolidate_count\((\d+)\)$`)
)

// NewControl returns a new control client
func NewControl(o Options) *Control {
	return &Control{
		client: newClient("control", o),
	}
}

// Trigger triggers an action
func (c *Control) Trigger(r TriggerRequest) error {
	return c.pool.Exec(func(ch pool.Channel) error {
		msg := fmt.Sprintf("TRIGGER %s", r.Action)
		if r.Data != "" {
			msg = fmt.Sprintf("%s %s", msg, r.Data)
		}

		err := ch.Write(msg)
		if err != nil {
			return err
		}

		// OK
		_, err = ch.Read()
		return err
	})
}

// Info returns server information
func (c *Control) Info() (InfoResponse, error) {
	res, err := c.pool.Query(func(ch pool.Channel) (interface{}, error) {
		err := ch.Write("INFO")
		if err != nil {
			return "", err
		}

		return ch.Read()
	})
	if err != nil {
		return InfoResponse{}, err
	}

	strs := infoRegexp.FindStringSubmatch(res.(string))
	if len(strs) != 9 {
		return InfoResponse{}, ErrInvalidInfoResponse
	}

	ints := make([]int, len(strs)-1, len(strs)-1)
	for idx, s := range strs[1:] {
		i, err := strconv.Atoi(s)
		if err != nil {
			return InfoResponse{}, ErrInvalidInfoResponse
		}

		ints[idx] = i
	}

	return InfoResponse{
		Uptime:              time.Duration(ints[0]) * time.Second,
		ClientsConnected:    ints[1],
		CommandsTotal:       ints[2],
		CommandLatencyBest:  time.Duration(ints[3]) * time.Millisecond,
		CommandLatencyWorst: time.Duration(ints[4]) * time.Millisecond,
		KVOpenCount:         ints[5],
		FSTOpenCount:        ints[6],
		FSTConsolidateCount: ints[7],
	}, nil
}
