// counter_test.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	srv, err := NewServer("tcp", "localhost:2380")
	if err != nil {
		t.Fatal(err)
	}

	go srv.Serve()
	defer srv.Close()

	for i := 0; i < 5; i++ {
		c, err := net.Dial("tcp", "localhost:2380")
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close()

		line, err := bufio.NewReader(c).ReadString('\n')
		if err != nil || !strings.Contains(line, fmt.Sprintf("#%d ", i+1)) {
			t.Errorf("invalid text received: %q (err:%v)", line, err)
			return
		}
	}
}
