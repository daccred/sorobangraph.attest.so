package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stellar/go/network"

	"github.com/daccred/sorobangraph.attest.so/controllers"
	"github.com/daccred/sorobangraph.attest.so/db"
	"github.com/daccred/sorobangraph.attest.so/handlers"
	"github.com/daccred/sorobangraph.attest.so/server"
)

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	databaseURL := getEnv("DATABASE_URL", "postgres://user:password@localhost/stellar_ingester?sslmode=disable")
	dbConn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	ingCfg := &handlers.Config{
		NetworkPassphrase:     getEnv("NETWORK_PASSPHRASE", network.TestNetworkPassphrase),
		CaptiveCoreConfigPath: getEnv("CAPTIVE_CORE_CONFIG_PATH", ""),
		CaptiveCoreBinaryPath: getEnv("CAPTIVE_CORE_BINARY_PATH", ""),
		HistoryArchiveURLs:    []string{getEnv("HISTORY_ARCHIVE_URLS", "https://history.stellar.org/prd/core-testnet/core_testnet_001")},
		StartLedger:           uint32(getEnvInt("START_LEDGER", 0)),
		EndLedger:             uint32(getEnvInt("END_LEDGER", 0)),
		EnableWebSocket:       getEnv("ENABLE_WEBSOCKET", "true") == "true",
		LogLevel:              getEnv("LOG_LEVEL", "info"),
	}

	logger := logrus.WithField("service", "ingester")
	ing, err := handlers.NewIngester(ingCfg, dbConn, logger)
	if err != nil {
		log.Fatalf("failed to create ingester: %v", err)
	}
	if err := ing.Start(context.Background()); err != nil {
		log.Fatalf("failed to start ingester: %v", err)
	}

	ctl := controllers.NewIngesterController(dbConn, ing.Stats())
	r := server.NewRouter(ctl)

	s := &server.Server{}
	if err := s.Run(r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}