package server

import (
	"os"
)

type Server struct{}

func (s *Server) Run(runner interface{ Run(addr ...string) error }) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return runner.Run(":" + port)
}
