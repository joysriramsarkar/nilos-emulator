// Package jsonwire implements the Content-Length framed JSON message channel
// used by both the Language Server Protocol (LSP) and the Debug Adapter
// Protocol (DAP).
//
// Framing:
//
//	Content-Length: 42\r\n
//	\r\n
//	{"jsonrpc":"2.0",...}
package jsonwire

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Conn is a framed JSON message channel.
type Conn struct {
	r  *bufio.Reader
	w  io.Writer
	mu sync.Mutex
}

// NewConn wraps a reader/writer.
func NewConn(r io.Reader, w io.Writer) *Conn {
	return &Conn{r: bufio.NewReader(r), w: w}
}

// ReadMessage returns the next message payload.
func (c *Conn) ReadMessage() ([]byte, error) {
	length := -1
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(line[len("content-length:"):])
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("jsonwire: bad Content-Length %q", v)
			}
			length = n
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("jsonwire: missing Content-Length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// Read decodes the next message into a generic map.
func (c *Conn) Read() (map[string]any, error) {
	buf, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Write encodes v and sends it with the framing header.
func (c *Conn) Write(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var hdr bytes.Buffer
	fmt.Fprintf(&hdr, "Content-Length: %d\r\n\r\n", len(payload))
	if _, err := c.w.Write(hdr.Bytes()); err != nil {
		return err
	}
	_, err = c.w.Write(payload)
	return err
}
