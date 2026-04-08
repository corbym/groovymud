package net

import (
	"errors"
	"fmt"
	"strings"

	"github.com/corbym/groovymud/gomud/internal/auth"
)

// loginLoop handles the full authentication flow for a session.
// It returns nil once the player is successfully authenticated.
func (srv *Server) loginLoop(s *Session) error {
	// Wait for the initial ENTER press after the splash.
	if _, err := s.readLine(); err != nil {
		return err
	}

	for {
		// Ask for username.
		if err := s.write("Enter your name (or 'new' to create an account): "); err != nil {
			return err
		}
		username, err := s.readLine()
		if err != nil {
			return err
		}
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}

		// Determine flow: new or existing player.
		if strings.EqualFold(username, "new") || !auth.PlayerExists(username) {
			if strings.EqualFold(username, "new") {
				if err := s.write("Choose a name: "); err != nil {
					return err
				}
				username, err = s.readLine()
				if err != nil {
					return err
				}
				username = strings.TrimSpace(username)
				if username == "" {
					continue
				}
			}

			if err := srv.registerFlow(s, username); err != nil {
				if errors.Is(err, errRetry) {
					continue
				}
				return err
			}
			return nil
		}

		// Existing player login.
		if err := srv.loginFlow(s, username); err != nil {
			if errors.Is(err, errRetry) {
				continue
			}
			return err
		}
		return nil
	}
}

var errRetry = errors.New("retry")

// registerFlow handles new-account creation.
func (srv *Server) registerFlow(s *Session, username string) error {
	s.suppressEcho()
	defer s.restoreEcho()

	if err := s.write("Choose a password: "); err != nil {
		return err
	}
	pass, err := s.readLine()
	if err != nil {
		return err
	}
	_ = s.writeLine("") // newline after hidden input

	if err := s.write("Confirm password: "); err != nil {
		return err
	}
	confirm, err := s.readLine()
	if err != nil {
		return err
	}
	_ = s.writeLine("")

	s.restoreEcho()

	if pass != confirm {
		_ = s.writeLine("Passwords do not match. Please try again.")
		return errRetry
	}

	player, err := auth.Register(username, pass)
	if err != nil {
		_ = s.writeLine(fmt.Sprintf("Registration failed: %v", err))
		return errRetry
	}

	s.mu.Lock()
	s.name = player.Name
	s.mu.Unlock()

	srv.registerSession(strings.ToLower(player.Name), s)
	_ = s.writeLine(fmt.Sprintf("Welcome, %s!", player.Name))
	return nil
}

// loginFlow handles existing-player authentication.
func (srv *Server) loginFlow(s *Session, username string) error {
	// Check duplicate login.
	existing := srv.FindSession(strings.ToLower(username))
	if existing != nil {
		if err := s.write("You are already logged in — throw the other copy out? (y/n): "); err != nil {
			return err
		}
		answer, err := s.readLine()
		if err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			_ = s.writeLine("Goodbye.")
			return fmt.Errorf("duplicate login declined")
		}
		// Kick the old session.
		_ = existing.writeLine("You have been disconnected because you logged in from another location.")
		_ = existing.conn.Close()
		srv.removeSession(existing)
	}

	s.suppressEcho()
	defer s.restoreEcho()

	if err := s.write("Password: "); err != nil {
		return err
	}
	pass, err := s.readLine()
	if err != nil {
		return err
	}
	_ = s.writeLine("")

	s.restoreEcho()

	player, err := auth.Login(username, pass)
	if err != nil {
		_ = s.writeLine("Invalid password. Please try again.")
		return errRetry
	}

	s.mu.Lock()
	s.name = player.Name
	s.mu.Unlock()

	srv.registerSession(strings.ToLower(player.Name), s)
	_ = s.writeLine(fmt.Sprintf("Welcome, %s!", player.Name))
	return nil
}
