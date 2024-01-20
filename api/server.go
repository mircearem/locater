package api

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/mircearem/locater/geo"
)

type Server struct {
	loc        *geo.Server
	e          *echo.Echo
	listenAddr string
}

func NewServer(listenAddr string, ctx context.Context) *Server {
	return &Server{
		loc:        geo.NewServer(ctx),
		e:          echo.New(),
		listenAddr: listenAddr,
	}
}

func (s *Server) Run() {

	// Run echo server

}
