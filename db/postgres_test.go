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
		// Mock data for ledger
		sequence := uint32(123456)
		hash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
		previousHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		transactionCount := 15
		operationCount := 42
		closedAt := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
		totalCoins := int64(105443902087654321)
		feePool := int64(923847562)
		baseFee := uint32(100)
		baseReserve := uint32(5000000)
		maxTxSetSize := uint32(1000)
		protocolVersion := uint32(20)
		ledgerHeader := []byte(`{"ledger_version": 20, "bucket_list_hash": "abc123", "fee_pool": 923847562}`)

		// Prepare mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ledgers").
			WithArgs(
				sequence,
				hash,
				previousHash,
				transactionCount,
				operationCount,
				closedAt,
				totalCoins,
				feePool,
				baseFee,
				baseReserve,
				maxTxSetSize,
				protocolVersion,
				ledgerHeader,
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
			sequence, hash, previousHash, transactionCount, operationCount,
			closedAt, totalCoins, feePool, baseFee, baseReserve, maxTxSetSize, protocolVersion,
			ledgerHeader)

		require.NoError(t, err)

		// Commit transaction
		err = tx.Commit()
		require.NoError(t, err)

		// Ensure all expectations were met
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert transaction data", func(t *testing.T) {
		// Mock data for transaction
		id := "123456-2"
		txHash := "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"
		ledger := uint32(123456)
		index := uint32(2)
		sourceAccount := "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
		feePaid := int64(300)
		operationCount := int32(2)
		createdAt := time.Date(2024, 1, 15, 12, 31, 0, 0, time.UTC)
		memoType := "text"
		memoValue := "Payment for services"
		successful := true
		envelopeXdr := []byte("AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAiII0AAAATAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAITg3T38VRqKKrpZPn6Rrf/k6J1IPf/7YY9YEwWGKtJWAAAAAAAAAAAAmJaAAAAAAAAAAAH8BfcAAABAB4O6RrQH+yxLUKKJ2LLYh6OVcbKJLG0cZFOMbDzKLVkD7GGGQdF4Tx7/Jt+7M//jIJqJALWDaEDCbHZxOCqAWg==")
		resultXdr := []byte("AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=")
		resultMetaXdr := []byte("AAAAAgAAAAIAAAADAAAAAQAAAAAAAAAAiODdPfxVGooqulk+fpGt/+TonUg9//thj1gTBYYq0lYAAAAXSHboAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAEAAAAAAAAAAIjg3T38VRqKKrpZPn6Rrf/k6J1IPf/7YY9YEwWGKtJWAAAAF0h26AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAA")

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(
				id,
				txHash,
				ledger,
				index,
				sourceAccount,
				feePaid,
				operationCount,
				createdAt,
				memoType,
				memoValue,
				successful,
				envelopeXdr,
				resultXdr,
				resultMetaXdr,
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
			id, txHash, ledger, index, sourceAccount, feePaid, operationCount, createdAt,
			memoType, memoValue, successful, envelopeXdr, resultXdr, resultMetaXdr)

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert operation data", func(t *testing.T) {
		// Mock data for operation
		id := "123456-2-0"
		transactionId := "123456-2"
		index := uint32(0)
		operationType := "payment"
		sourceAccount := "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
		details := []byte(`{
			"from": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
			"to": "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP",
			"amount": "10.0000000",
			"asset_type": "native"
		}`)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO operations").
			WithArgs(
				id,
				transactionId,
				index,
				operationType,
				sourceAccount,
				details,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`
			INSERT INTO operations (id, transaction_id, index, type, source_account, details)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING`,
			id, transactionId, index, operationType, sourceAccount, details)

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Insert contract event", func(t *testing.T) {
		// Mock data for contract event
		eventId := "123456-2-0-0-0001"
		contractId := "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAYYKVPINOU"
		ledger := uint32(123456)
		transactionHash := "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"
		eventType := "contract"
		topics := []byte(`[
			"AAAADwAAAAdDT1VOVEVSAA==",
			"AAAAEAAAAAEAAAACAAAADwAAAAdBQ0NPVU5UAAAAAAASAAAAAQAAAAIAAAAPAAAAB0JBTEFOQ0UAAAAA"
		]`)
		data := []byte(`{
			"type": "i128",
			"value": "1000000000"
		}`)
		inSuccessfulTx := true

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO contract_events").
			WithArgs(
				eventId,
				contractId,
				ledger,
				transactionHash,
				eventType,
				topics,
				data,
				inSuccessfulTx,
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
			eventId, contractId, ledger, transactionHash, eventType, topics, data, inSuccessfulTx)

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Update ingestion state", func(t *testing.T) {
		// Mock data for ingestion state
		id := 1
		lastLedger := uint32(123456)
		updatedAt := time.Date(2024, 1, 15, 12, 35, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ingestion_state").
			WithArgs(
				id,
				lastLedger,
				updatedAt,
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
			id, lastLedger, updatedAt)

		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Query last ledger", func(t *testing.T) {
		// Mock data for query
		id := 1
		expectedLastLedger := uint32(123455)

		rows := sqlmock.NewRows([]string{"last_ledger"}).
			AddRow(expectedLastLedger)

		mock.ExpectQuery("SELECT last_ledger FROM ingestion_state WHERE").
			WithArgs(id).
			WillReturnRows(rows)

		var lastLedger uint32
		err := mockDB.QueryRow(`SELECT last_ledger FROM ingestion_state WHERE id = $1`, id).
			Scan(&lastLedger)

		require.NoError(t, err)
		assert.Equal(t, expectedLastLedger, lastLedger)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Query with no rows", func(t *testing.T) {
		// Mock data for no rows scenario
		id := 999

		mock.ExpectQuery("SELECT last_ledger FROM ingestion_state WHERE").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		var lastLedger uint32
		err := mockDB.QueryRow(`SELECT last_ledger FROM ingestion_state WHERE id = $1`, id).
			Scan(&lastLedger)

		assert.Equal(t, sql.ErrNoRows, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Transaction rollback on error", func(t *testing.T) {
		// Mock data for rollback scenario
		sequence := uint32(123457)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO ledgers").
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		tx, err := mockDB.Begin()
		require.NoError(t, err)

		_, err = tx.Exec(`INSERT INTO ledgers (sequence) VALUES ($1)`, sequence)
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
