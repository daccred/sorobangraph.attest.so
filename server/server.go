package server

import "github.com/daccred/sorobangraph.attest.so/config"

func Init() {
	config := config.GetConfig()
	r := NewRouter()
	r.Run(config.GetString("server.address"))
}
