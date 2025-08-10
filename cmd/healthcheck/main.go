package main

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/network"

	"github.com/daccred/sorobangraph.attest.so/controllers"
	"github.com/daccred/sorobangraph.attest.so/db"
	"github.com/daccred/sorobangraph.attest.so/handlers"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://postgres:UikJouInuaAtzDYMdpsOlFxWPORldLBP@turntable.proxy.rlwy.net:52543/railway"
	}

	log.Println("Testing database connection...")
	dbConn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("✅ Database connection successful!")

	log.Println("Testing ingester creation...")
	ingCfg := &handlers.Config{
		NetworkPassphrase:  network.TestNetworkPassphrase,
		HistoryArchiveURLs: []string{"https://history.stellar.org/prd/core-testnet/core_testnet_001"},
		StartLedger:        0,
		EndLedger:          0,
		EnableWebSocket:    true,
		LogLevel:           "info",
	}

	logger := logrus.WithField("service", "ingester")
	ing, err := handlers.NewIngester(ingCfg, dbConn, logger)
	if err != nil {
		log.Fatalf("failed to create ingester: %v", err)
	}
	log.Println("✅ Ingester created successfully!")

	log.Println("Testing controller creation...")
	ctl := controllers.NewIngesterController(dbConn, ing.Stats())
	if ctl == nil {
		log.Fatalf("failed to create controller")
	}
	log.Println("✅ Controller created successfully!")

	log.Println("Testing basic database queries...")

	// Test querying the ingestion_state table
	var lastLedger uint32
	err = dbConn.QueryRow("SELECT last_ledger FROM ingestion_state WHERE id = 1").Scan(&lastLedger)
	if err != nil {
		log.Fatalf("failed to query ingestion_state: %v", err)
	}
	log.Printf("✅ Current ingestion state: last_ledger = %d", lastLedger)

	// Test inserting a test ledger (then delete it)
	testSeq := uint32(999999)
	_, err = dbConn.Exec(`
		INSERT INTO ledgers (sequence, hash, previous_hash, transaction_count, 
			operation_count, closed_at, total_coins, fee_pool, base_fee, 
			base_reserve, max_tx_set_size, protocol_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (sequence) DO NOTHING`,
		testSeq, "test_hash", "test_prev_hash", 0, 0, time.Now(),
		0, 0, 100, 5000000, 1000, 21)

	if err != nil {
		log.Fatalf("failed to insert test ledger: %v", err)
	}

	// Clean up test data
	_, err = dbConn.Exec("DELETE FROM ledgers WHERE sequence = $1", testSeq)
	if err != nil {
		log.Printf("Warning: failed to clean up test ledger: %v", err)
	}

	log.Println("✅ Database operations successful!")
	log.Println("\n🎉 All tests passed! Your ingester is ready to run.")
	log.Println("\nNext steps:")
	log.Println("1. Run: make build")
	log.Println("2. Run: ./sorobangraph.attest.so")
	log.Println("3. Visit: http://localhost:8080/health")
}
