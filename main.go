package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stellar/go/network"

	"github.com/daccred/sorobangraph.attest.so/config"
	"github.com/daccred/sorobangraph.attest.so/controllers"
	"github.com/daccred/sorobangraph.attest.so/db"
	"github.com/daccred/sorobangraph.attest.so/handlers"
	"github.com/daccred/sorobangraph.attest.so/server"
	"github.com/subosito/gotenv"
)

func main() {
	// Load environment variables from .env if present
	_ = gotenv.Load()

	// Parse environment flag (default to development)
	env := flag.String("e", "development", "application environment (development|production|test)")
	flag.Parse()

	// Initialize config based on environment
	config.Init(*env)
	cfg := config.GetConfig()

	// Set Gin mode from env/config
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = cfg.GetString("server.gin_mode")
	}
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Resolve database URL from env or config, with fallback
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		raw := cfg.GetString("database.url")
		expanded := os.ExpandEnv(raw)
		// If expansion didn't resolve (still contains ${...}) or is empty, keep databaseURL empty to allow fallback
		if expanded != "" && !strings.Contains(expanded, "${") {
			databaseURL = expanded
		}
	}
	if databaseURL == "" {
		databaseURL = "postgres://user:password@localhost/stellar_ingester?sslmode=disable"
	}

	dbConn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Parse filter contracts from environment variable or config
	filterContractsEnv := getEnv("FILTER_CONTRACTS", "")
	var filterContracts []string
	if filterContractsEnv != "" {
		// Environment variable takes precedence
		filterContracts = strings.Split(filterContractsEnv, ",")
		for i := range filterContracts {
			filterContracts[i] = strings.TrimSpace(filterContracts[i])
		}
	} else {
		// Read from config file
		filterContracts = cfg.GetStringSlice("stellar.filter_contracts")
	}

	ingCfg := &handlers.Config{
		NetworkPassphrase:     getEnv("NETWORK_PASSPHRASE", network.TestNetworkPassphrase),
		CaptiveCoreConfigPath: getEnv("CAPTIVE_CORE_CONFIG_PATH", cfg.GetString("captive_core.config_path")),
		CaptiveCoreBinaryPath: getEnv("CAPTIVE_CORE_BINARY_PATH", cfg.GetString("captive_core.binary_path")),
		HistoryArchiveURLs:    []string{getEnv("HISTORY_ARCHIVE_URLS", "https://history.stellar.org/prd/core-testnet/core_testnet_001")},
		StartLedger:           uint32(getEnvInt("START_LEDGER", cfg.GetInt("stellar.start_ledger"))),
		EndLedger:             uint32(getEnvInt("END_LEDGER", cfg.GetInt("stellar.end_ledger"))),
		EnableWebSocket:       getEnv("ENABLE_WEBSOCKET", "true") == "true",
		LogLevel:              getEnv("LOG_LEVEL", cfg.GetString("logging.level")),
		FilterContracts:       filterContracts,
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
