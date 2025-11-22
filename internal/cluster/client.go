package cluster

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
)

func SendTask(ctx context.Context, addr string, task *RecTask) (*RecResponse, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	if err := enc.Encode(task); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bufio.NewReader(conn))
	var resp RecResponse
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
