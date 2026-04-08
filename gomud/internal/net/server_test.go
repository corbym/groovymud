package net_test

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	mudnet "github.com/corbym/groovymud/gomud/internal/net"
	"github.com/corbym/groovymud/gomud/internal/store"
)

func startServer(t *testing.T) string {
	t.Helper()
	if err := store.Open("file::memory:?cache=shared&_busy_timeout=5000"); err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(store.Close)
	srv := mudnet.NewServer()
	if err := srv.Listen(":0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(srv.Close)
	go srv.Serve()
	return srv.Addr().String()
}

func TestSplashBanner(t *testing.T) {
	addr := startServer(t)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(conn)

	var sb strings.Builder
	for {
		line, err := r.ReadString('\n')
		sb.WriteString(line)
		if strings.Contains(line, "ENTER") {
			break
		}
		if err != nil {
			break
		}
	}
	banner := sb.String()
	if !strings.Contains(banner, "GroovyMud") {
		t.Errorf("expected GroovyMud in banner, got: %q", banner)
	}
	if !strings.Contains(banner, "ENTER") {
		t.Errorf("expected ENTER prompt in banner")
	}
}
