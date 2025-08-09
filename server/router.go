package server

import (
	"time"

	"github.com/daccred/sorobangraph.attest.so/controllers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(ingesterController *controllers.IngesterController) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	cfg := cors.DefaultConfig()
	cfg.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	cfg.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	cfg.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	cfg.AllowCredentials = true
	cfg.MaxAge = 12 * time.Hour
	r.Use(cors.New(cfg))

	ingesterController.RegisterRoutes(r)

	return r
}
