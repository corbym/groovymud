package net

import (
	"fmt"
	"log"
	"net"
	"sync"
)

const splash = `
 ██████╗ ██████╗  ██████╗  ██████╗ ██╗   ██╗██╗   ██╗███╗   ███╗██╗   ██╗██████╗ 
██╔════╝ ██╔══██╗██╔═══██╗██╔═══██╗██║   ██║╚██╗ ██╔╝████╗ ████║██║   ██║██╔══██╗
██║  ███╗██████╔╝██║   ██║██║   ██║██║   ██║ ╚████╔╝ ██╔████╔██║██║   ██║██║  ██║
██║   ██║██╔══██╗██║   ██║██║   ██║╚██╗ ██╔╝  ╚██╔╝  ██║╚██╔╝██║██║   ██║██║  ██║
╚██████╔╝██║  ██║╚██████╔╝╚██████╔╝ ╚████╔╝    ██║   ██║ ╚═╝ ██║╚██████╔╝██████╔╝
 ╚═════╝ ╚═╝  ╚═╝ ╚═════╝  ╚═════╝   ╚═══╝     ╚═╝   ╚═╝     ╚═╝ ╚═════╝ ╚═════╝ 
                          A Multi-User Dungeon Adventure
                              Welcome to GroovyMud!
`

// Server is a TCP server that accepts MUD client connections.
type Server struct {
	listener net.Listener
	mu       sync.RWMutex
	sessions map[string]*Session // keyed by player name (lower-case) once logged in

	// Hook called after login succeeds; set by higher layers.
	OnLogin func(s *Session)
	// Hook called when a player issues an in-game command.
	OnCommand func(s *Session, line string)
}

// NewServer creates a new Server but does not start listening.
func NewServer() *Server {
	return &Server{
		sessions: make(map[string]*Session),
	}
}

// Listen starts the TCP listener on addr (e.g. ":2222").
func (srv *Server) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	srv.listener = ln
	return nil
}

// Addr returns the listener's address (useful when port is 0).
func (srv *Server) Addr() net.Addr {
	if srv.listener == nil {
		return nil
	}
	return srv.listener.Addr()
}

// Serve accepts connections in a loop until the listener is closed.
func (srv *Server) Serve() {
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			return // listener closed
		}
		sess := newSession(conn, srv)
		go srv.handleSession(sess)
	}
}

// Close shuts down the listener.
func (srv *Server) Close() {
	if srv.listener != nil {
		_ = srv.listener.Close()
	}
}

// handleSession runs the per-connection lifecycle.
func (srv *Server) handleSession(s *Session) {
	defer func() {
		_ = s.conn.Close()
		srv.removeSession(s)
	}()

	if err := s.sendSplash(); err != nil {
		log.Printf("splash: %v", err)
		return
	}

	if err := srv.loginLoop(s); err != nil {
		log.Printf("login: %v", err)
		return
	}

	// Signal higher layers that login succeeded.
	if srv.OnLogin != nil {
		srv.OnLogin(s)
	}

	// Main command loop.
	for {
		_ = s.write("> ")
		line, err := s.readLine()
		if err != nil {
			return
		}
		if srv.OnCommand != nil {
			srv.OnCommand(s, line)
		}
	}
}

// sendSplash writes the ASCII art banner and the "Press ENTER" prompt.
func (s *Session) sendSplash() error {
	if err := s.write(splash); err != nil {
		return err
	}
	return s.writeLine("Press ENTER to continue...")
}

// registerSession adds a logged-in session to the server map.
func (srv *Server) registerSession(name string, s *Session) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.sessions[name] = s
}

// removeSession removes a session from the server map.
func (srv *Server) removeSession(s *Session) {
	name := s.Name()
	if name == "" {
		return
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.sessions[name] == s {
		delete(srv.sessions, name)
	}
}

// FindSession returns an active session by player name, or nil.
func (srv *Server) FindSession(name string) *Session {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	return srv.sessions[name]
}
