package net

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

// IAC telnet command byte
const IAC = 0xFF

// Telnet option commands
const (
	WILL = 0xFB
	WONT = 0xFC
	DO   = 0xFD
	DONT = 0xFE
)

// Telnet options
const (
	OPT_ECHO = 0x01
)

// SessionState tracks where in the login/play flow a session is.
type SessionState int

const (
	StateWelcome SessionState = iota
	StateAskUsername
	StateAskPassword
	StateAskNewPassword
	StateAskConfirmPassword
	StateAskKickDuplicate
	StatePlaying
)

// Session represents a single client connection.
type Session struct {
	mu       sync.Mutex
	conn     net.Conn
	reader   *bufio.Reader
	state    SessionState
	name     string
	tempPass string // used during registration confirm

	// Server back-reference for duplicate-login handling and world access
	server *Server
}

func newSession(conn net.Conn, srv *Server) *Session {
	return &Session{
		conn:   conn,
		reader: bufio.NewReader(conn),
		state:  StateWelcome,
		server: srv,
	}
}

// writeLine sends a line with \r\n telnet line ending (internal use).
func (s *Session) writeLine(msg string) error {
	_, err := fmt.Fprintf(s.conn, "%s\r\n", msg)
	return err
}

// WriteLine is the exported version for use by the engine.
func (s *Session) WriteLine(msg string) error {
	return s.writeLine(msg)
}

// write sends raw bytes (no newline added).
func (s *Session) write(msg string) error {
	_, err := fmt.Fprint(s.conn, msg)
	return err
}

// sendIAC sends a 3-byte telnet IAC sequence.
func (s *Session) sendIAC(cmd, opt byte) error {
	_, err := s.conn.Write([]byte{IAC, cmd, opt})
	return err
}

// suppressEcho tells the telnet client to not echo characters (for password input).
func (s *Session) suppressEcho() {
	_ = s.sendIAC(WILL, OPT_ECHO)
}

// restoreEcho tells the telnet client to echo characters again.
func (s *Session) restoreEcho() {
	_ = s.sendIAC(WONT, OPT_ECHO)
}

// readLine reads a line, stripping telnet IAC sequences and trailing \r\n.
func (s *Session) readLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	line = stripIAC(line)
	return line, nil
}

// stripIAC removes telnet IAC sequences from a string.
func stripIAC(s string) string {
	b := []byte(s)
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == IAC {
			// IAC sequences: IAC <cmd> <opt> (3 bytes) or IAC IAC (escaped 0xFF)
			if i+1 < len(b) {
				next := b[i+1]
				if next == IAC {
					out = append(out, 0xFF)
					i++
				} else if next == WILL || next == WONT || next == DO || next == DONT {
					i += 2 // skip cmd + opt
				} else {
					i++ // skip single-byte IAC command
				}
			}
			continue
		}
		out = append(out, b[i])
	}
	return string(out)
}

// Name returns the player name for this session.
func (s *Session) Name() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name
}
