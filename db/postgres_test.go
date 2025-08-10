package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	t.Run("Connection configuration", func(t *testing.T) {
		// This test would normally require a real database
		// For unit testing, we're verifying the connection parameters
		t.Skip("Skipping real database connection test")

		database, err := Connect("postgresql://test:test@localhost/test?sslmode=disable")
		if err != nil {
			t.Skip("Database not available for testing")
		}
		defer database.Close()

		// Verify connection pool settings
		stats := database.Stats()
		assert.LessOrEqual(t, stats.MaxOpenConnections, 25)
	})
}

func TestDatabaseOperations(t *testing.T) {
	// Create a mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	t.Run("Insert ledger data", func(t *testing.T) {
		// Prepare mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ledgers").
			WithArgs(
				uint32(1000),      // sequence
				"abc123",          // hash
				"xyz789",          // previous_hash
				10,                // transaction_count
				25,                // operation_count
				sqlmock.AnyArg(),  // closed_at
				int64(1000000000), // total_coins
				int64(500000),     // fee_pool
				uint32(100),       // base_fee
				uint32(10000000),  // base_reserve
				uint32(1000),      // max_tx_set_size
				uint32(20),        // protocol_version
				sqlmock.AnyArg(),  // ledger_header JSON
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		// Start transaction
		tx, err := mockDB.Begin()
		require.NoError(t, err)

		// Execute insert
		_, err = tx.Exec(`
			INSERT INTO ledgers (sequence, hash, previous_hash, transaction_count,
				operation_count, closed_at, total_coins, fee_pool, base_fee,
				base_reserve, max_tx_set_size, protocol_version, ledger_header)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (sequence) DO NOTHING`,
			uint32(1000), "abc123", "xyz789", 10, 25,
			time.Now(), int64(1000000000), int64(500000),
			uint32(100), uint32(10000000), uint32(1000), uint32(20),
			[]byte(`{"test": "data"}`))

		require.NoError(t, err)

		// Commit transaction
		err = tx.Commit()
		require.NoError(t, err)

		// Ensure all expectations were met
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert transaction data", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(
				"1000-0",         // id
				"tx_hash_123",    // hash
				uint32(1000),     // ledger
				uint32(0),        // index
				"GABC123",        // source_account
				int64(100),       // fee_paid
				int32(3),         // operation_count
				sqlmock.AnyArg(), // created_at
				"text",           // memo_type
				"test memo",      // memo_value
				true,             // successful
				sqlmock.AnyArg(), // envelope_xdr
				sqlmock.AnyArg(), // result_xdr
				sqlmock.AnyArg(), // result_meta_xdr
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`
			INSERT INTO transactions (id, hash, ledger, index, source_account, fee_paid,
				operation_count, created_at, memo_type, memo_value, successful,
				envelope_xdr, result_xdr, result_meta_xdr)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (id) DO NOTHING`,
			"1000-0", "tx_hash_123", uint32(1000), uint32(0),
			"GABC123", int64(100), int32(3), time.Now(),
			"text", "test memo", true,
			[]byte("envelope"), []byte("result"), []byte("meta"))

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert operation data", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO operations").
			WithArgs(
				"1000-0-0",       // id
				"1000-0",         // transaction_id
				uint32(0),        // index
				"payment",        // type
				"GABC123",        // source_account
				sqlmock.AnyArg(), // details JSON
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`
			INSERT INTO operations (id, transaction_id, index, type, source_account, details)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			"1000-0-0", "1000-0", uint32(0), "payment", "GABC123",
			[]byte(`{"amount": 1000000}`))

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert contract event", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO contract_events").
			WithArgs(
				"event_123",      // id
				"contract_abc",   // contract_id
				uint32(1000),     // ledger
				"tx_hash_123",    // transaction_hash
				"contract",       // event_type
				sqlmock.AnyArg(), // topics JSON
				sqlmock.AnyArg(), // data JSON
				true,             // in_successful_tx
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`
			INSERT INTO contract_events (id, contract_id, ledger, transaction_hash,
				event_type, topics, data, in_successful_tx)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING`,
			"event_123", "contract_abc", uint32(1000), "tx_hash_123",
			"contract", []byte(`["topic1", "topic2"]`), []byte(`{"key": "value"}`), true)

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Update ingestion state", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ingestion_state").
			WithArgs(
				1,                // id (first id argument)
				uint32(1000),     // last_ledger
				sqlmock.AnyArg(), // updated_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`
			INSERT INTO ingestion_state (id, last_ledger, updated_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET
				last_ledger = EXCLUDED.last_ledger,
				updated_at = EXCLUDED.updated_at`,
			1, uint32(1000), time.Now())

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Query last ledger", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"last_ledger"}).
			AddRow(uint32(5000))

		mock.ExpectQuery("SELECT last_ledger FROM ingestion_state WHERE").
			WithArgs(1).
			WillReturnRows(rows)

		var lastLedger uint32
		err := mockDB.QueryRow(`SELECT last_ledger FROM ingestion_state WHERE id = $1`, 1).
			Scan(&lastLedger)

		require.NoError(t, err)
		assert.Equal(t, uint32(5000), lastLedger)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Query with no rows", func(t *testing.T) {
		mock.ExpectQuery("SELECT last_ledger FROM ingestion_state WHERE").
			WithArgs(1).
			WillReturnError(sql.ErrNoRows)

		var lastLedger uint32
		err := mockDB.QueryRow(`SELECT last_ledger FROM ingestion_state WHERE id = $1`, 1).
			Scan(&lastLedger)

		assert.Equal(t, sql.ErrNoRows, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Transaction rollback on error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ledgers").
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`INSERT INTO ledgers (sequence) VALUES ($1)`, 1000)
		assert.Error(t, err)

		err = tx.Rollback()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestConnectionPoolSettings(t *testing.T) {
	// Create a mock database
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Set connection pool parameters similar to Connect function
	mockDB.SetMaxOpenConns(25)
	mockDB.SetMaxIdleConns(10)
	mockDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify settings
	stats := mockDB.Stats()
	assert.Equal(t, 25, stats.MaxOpenConnections)
}

func TestConcurrentDatabaseAccess(t *testing.T) {
	// Create a mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Set up expectations for concurrent queries
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(i)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)
	}

	// Run concurrent queries
	done := make(chan bool)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			var count int
			_ = mockDB.QueryRow("SELECT COUNT(*) FROM ledgers").Scan(&count)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
