// Package e2e contains outside-in approval tests for GroovyMud.
//
// Each test starts a real TCP server backed by an in-memory SQLite database
// and the full TOML world loaded from ../../world/.  A scripted telnet session
// is driven over the connection and the resulting output is compared against a
// committed *.approved.txt file in ../approvals/.
//
// To re-baseline an approval after an intentional change, delete (or update)
// the corresponding *.approved.txt file and re-run the tests.
package e2e_test

import (
	"bufio"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/corbym/groovymud/gomud/internal/approval"
	"github.com/corbym/groovymud/gomud/internal/engine"
	"github.com/corbym/groovymud/gomud/internal/loader"
	"github.com/corbym/groovymud/gomud/internal/store"
	"github.com/corbym/groovymud/gomud/internal/world"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testServer starts a fully-wired server on a random port and returns its
// address. The server and in-memory DB are automatically torn down when the
// test ends.
func testServer(t *testing.T) string {
	t.Helper()

	// Each test gets its own in-memory SQLite instance.
	dsn := "file::memory:?cache=shared&_busy_timeout=5000"
	if err := store.Open(dsn); err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(store.Close)

	// Load the real world TOML data.
	w := world.New()
	_, thisFile, _, _ := runtime.Caller(0)
	worldDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "world")
	if err := loader.LoadDir(w, worldDir); err != nil {
		t.Fatalf("loader.LoadDir: %v", err)
	}

	eng := engine.New(w)
	if err := eng.Server.Listen(":0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(eng.Server.Close)
	go eng.Server.Serve()

	return eng.Server.Addr().String()
}

// approvalPath returns the path of the *.approved.txt file for the given test
// scenario name.
func approvalPath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "approvals", name+".approved.txt")
}

// ---------------------------------------------------------------------------
// session helper — thin wrapper around a TCP connection.
// ---------------------------------------------------------------------------

type session struct {
	conn   net.Conn
	reader *bufio.Reader
}

func dial(t *testing.T, addr string) *session {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &session{conn: conn, reader: bufio.NewReader(conn)}
}

// readUntilSuffix reads bytes until the accumulated output ends with suffix,
// or until the per-call deadline (5 s) expires.
func (s *session) readUntilSuffix(t *testing.T, suffix string) string {
	t.Helper()
	_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var sb strings.Builder
	for {
		b, err := s.reader.ReadByte()
		if err != nil {
			break
		}
		sb.WriteByte(b)
		if strings.HasSuffix(sb.String(), suffix) {
			break
		}
	}
	_ = s.conn.SetReadDeadline(time.Time{})
	return sb.String()
}

// readUntilPrompt reads until the "> " command prompt appears.
func (s *session) readUntilPrompt(t *testing.T) string {
	return s.readUntilSuffix(t, "> ")
}

// readUntil reads whole lines until stopFn matches, used only for the splash.
func (s *session) readUntil(t *testing.T, stopFn func(string) bool) string {
	t.Helper()
	_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var sb strings.Builder
	for {
		line, err := s.reader.ReadString('\n')
		if line != "" {
			sb.WriteString(line)
			if stopFn(line) {
				break
			}
		}
		if err != nil {
			break
		}
	}
	_ = s.conn.SetReadDeadline(time.Time{})
	return sb.String()
}

// send writes a line followed by \r\n (standard telnet line ending).
func (s *session) send(t *testing.T, line string) {
	t.Helper()
	_ = s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := s.conn.Write([]byte(line + "\r\n")); err != nil {
		t.Fatalf("send %q: %v", line, err)
	}
	_ = s.conn.SetWriteDeadline(time.Time{})
}

// normalise strips telnet IAC bytes and collapses \r\n → \n so approval
// files use plain Unix line endings.
func normalise(s string) string {
	// Remove IAC sequences (0xFF prefix, 3-byte commands).
	b := []byte(s)
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == 0xFF {
			if i+2 < len(b) {
				i += 2
			} else if i+1 < len(b) {
				i++
			}
			continue
		}
		out = append(out, b[i])
	}
	result := string(out)
	result = strings.ReplaceAll(result, "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "\n")
	return result
}

// quickRegister performs a full new-player registration without capturing output.
// It leaves the session in the playing state (command prompt received).
func quickRegister(t *testing.T, s *session, name, password string) {
	t.Helper()
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "ENTER") })
	s.send(t, "")
	s.readUntilSuffix(t, ": ") // "Enter your name..."
	s.send(t, "new")
	s.readUntilSuffix(t, ": ") // "Choose a name: "
	s.send(t, name)
	s.readUntilSuffix(t, ": ") // "Choose a password: "
	s.send(t, password)
	s.readUntilSuffix(t, ": ") // "Confirm password: "
	s.send(t, password)
	s.readUntilPrompt(t) // consume welcome + auto-look
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestSplashBanner verifies the text shown to a freshly-connected client.
func TestSplashBanner(t *testing.T) {
	addr := testServer(t)
	s := dial(t, addr)

	// Read until "Press ENTER" prompt.
	raw := s.readUntil(t, func(line string) bool {
		return strings.Contains(line, "ENTER")
	})
	got := normalise(raw)

	approval.Verify(t, approvalPath("splash_banner"), got)
}

// TestNewUserRegistration verifies the full registration flow for a brand-new
// player, ending with the auto-look output in Town Square.
func TestNewUserRegistration(t *testing.T) {
	addr := testServer(t)
	s := dial(t, addr)

	var sb strings.Builder

	// splash
	sb.WriteString(normalise(s.readUntil(t, func(l string) bool { return strings.Contains(l, "ENTER") })))

	// Press ENTER.
	s.send(t, "")

	// "Enter your name (or 'new' to create an account): "
	sb.WriteString(normalise(s.readUntilSuffix(t, ": ")))
	s.send(t, "new")

	// "Choose a name: "
	sb.WriteString(normalise(s.readUntilSuffix(t, ": ")))
	s.send(t, "ApprovalHero")

	// "Choose a password: " (preceded by IAC WILL ECHO bytes — stripped by normalise)
	sb.WriteString(normalise(s.readUntilSuffix(t, ": ")))
	s.send(t, "secret123")

	// "\r\nConfirm password: "
	sb.WriteString(normalise(s.readUntilSuffix(t, ": ")))
	s.send(t, "secret123")

	// welcome + auto-look + first "> " prompt
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	approval.Verify(t, approvalPath("new_user_registration"), sb.String())
}

// TestLookAfterLogin verifies the room description seen after logging in.
func TestLookAfterLogin(t *testing.T) {
	addr := testServer(t)
	s := dial(t, addr)

	quickRegister(t, s, "LookHero", "pass1234")

	s.send(t, "look")
	lookOutput := s.readUntilPrompt(t)

	approval.Verify(t, approvalPath("look_town_centre"), normalise(lookOutput))
}

// TestMovement verifies movement from Town Square to Town Square North.
func TestMovement(t *testing.T) {
	addr := testServer(t)
	s := dial(t, addr)

	// Register and log in.
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "ENTER") })
	s.send(t, "")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "name") })
	s.send(t, "new")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "Choose a name") })
	s.send(t, "MoveHero")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "password") })
	s.send(t, "pass1234")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "Confirm") })
	s.send(t, "pass1234")
	s.readUntilPrompt(t) // consume welcome + auto-look

	// Go north.
	s.send(t, "north")
	northOutput := s.readUntilPrompt(t)

	// Go back south.
	s.send(t, "south")
	southOutput := s.readUntilPrompt(t)

	combined := normalise(northOutput) + "---\n" + normalise(southOutput)
	approval.Verify(t, approvalPath("movement_north_south"), combined)
}

// TestItemsGetDrop verifies the full get → inventory → drop → look cycle.
func TestItemsGetDrop(t *testing.T) {
	addr := testServer(t)
	s := dial(t, addr)

	// Register and log in.
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "ENTER") })
	s.send(t, "")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "name") })
	s.send(t, "new")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "Choose a name") })
	s.send(t, "ItemHero")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "password") })
	s.send(t, "pass1234")
	s.readUntil(t, func(l string) bool { return strings.Contains(l, "Confirm") })
	s.send(t, "pass1234")
	s.readUntilPrompt(t) // welcome + auto-look

	var sb strings.Builder

	// get sword
	s.send(t, "get sword")
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	// inventory
	s.send(t, "inventory")
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	// drop sword
	s.send(t, "drop sword")
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	// inventory again — should be empty
	s.send(t, "inventory")
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	// look — sword should be back in room
	s.send(t, "look")
	sb.WriteString(normalise(s.readUntilPrompt(t)))

	approval.Verify(t, approvalPath("items_get_drop"), sb.String())
}
