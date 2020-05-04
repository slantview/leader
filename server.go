package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/slantview/leader/election"
)

// Server is the main server container for the service.
type Server struct {
	leaderElection election.Election
}

// NewServer returns a new Server.
func NewServer(e election.Election) *Server {
	return &Server{
		leaderElection: e,
	}
}

// Hello is the handler for the /hello url for server.
func (s *Server) Hello(c echo.Context) error {
	if s.leaderElection.IsLeader() {
		c.String(http.StatusOK, "Hello, World!")
		return nil
	}

	// If we are not the leader, do an HTTP 302 redirect to the leader.
	// NOTE(smfr): We are using an HTTP 302 here, but this could be a 301/307
	// depending on the circumstances of our redirection.
	c.Redirect(http.StatusFound, s.leaderElection.GetLeader())
	return nil
}

// HealthCheck is the handler for keepalive on the leader election keepalives.
// TODO(smfr): We may want to check for a connection to our backend with some
// sort of keepalive so we can return an error here.
func (s *Server) HealthCheck(c echo.Context) error {
	c.String(http.StatusOK, "ok")
	return nil
}
