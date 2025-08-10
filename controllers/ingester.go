package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/daccred/sorobangraph.attest.so/models"
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

type IngesterController struct {
	db    *sql.DB
	stats *models.Stats
}

func NewIngesterController(db *sql.DB, stats *models.Stats) *IngesterController {
	return &IngesterController{db: db, stats: stats}
}

func (ic *IngesterController) RegisterRoutes(r *gin.Engine) {
	store := persistence.NewInMemoryStore(time.Minute)

	r.GET("/health", ic.HealthCheck)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ledgers", ic.GetLedgers)
		v1.GET("/ledgers/:sequence", ic.GetLedger)
		v1.GET("/transactions", ic.GetTransactions)
		v1.GET("/transactions/:hash", ic.GetTransaction)
		v1.GET("/operations", ic.GetOperations)
		v1.GET("/contract-events", ic.GetContractEvents)
		v1.GET("/stats", cache.CachePage(store, time.Minute, ic.GetStats))
	}
}

func (ic *IngesterController) HealthCheck(c *gin.Context) {
	if err := ic.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "Database connection failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (ic *IngesterController) GetLedgers(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")

	rows, err := ic.db.Query(`
		SELECT sequence, hash, previous_hash, transaction_count, operation_count,
		       closed_at, protocol_version
		FROM ledgers
		ORDER BY sequence DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch ledgers"})
		return
	}
	defer rows.Close()

	var ledgers []models.LedgerInfo
	for rows.Next() {
		var ledger models.LedgerInfo
		if err := rows.Scan(&ledger.Sequence, &ledger.Hash, &ledger.PreviousHash,
			&ledger.TransactionCount, &ledger.OperationCount, &ledger.ClosedAt,
			&ledger.ProtocolVersion); err == nil {
			ledgers = append(ledgers, ledger)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": ledgers})
}

func (ic *IngesterController) GetLedger(c *gin.Context) {
	sequence := c.Param("sequence")
	var ledger models.LedgerInfo
	err := ic.db.QueryRow(`
		SELECT sequence, hash, previous_hash, transaction_count, operation_count,
		       closed_at, total_coins, fee_pool, base_fee, base_reserve,
		       max_tx_set_size, protocol_version
		FROM ledgers WHERE sequence = $1`, sequence).Scan(
		&ledger.Sequence, &ledger.Hash, &ledger.PreviousHash, &ledger.TransactionCount,
		&ledger.OperationCount, &ledger.ClosedAt, &ledger.TotalCoins, &ledger.FeePool,
		&ledger.BaseFee, &ledger.BaseReserve, &ledger.MaxTxSetSize, &ledger.ProtocolVersion)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Ledger not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch ledger"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ledger})
}

func (ic *IngesterController) GetTransactions(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")

	rows, err := ic.db.Query(`
		SELECT id, hash, ledger, index, source_account, fee_paid,
		       operation_count, created_at, memo_type, memo_value, successful
		FROM transactions
		ORDER BY ledger DESC, index DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch transactions"})
		return
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		var memoType, memoValue sql.NullString
		if err := rows.Scan(&tx.ID, &tx.Hash, &tx.Ledger, &tx.Index,
			&tx.SourceAccount, &tx.FeePaid, &tx.OperationCount,
			&tx.CreatedAt, &memoType, &memoValue, &tx.Successful); err == nil {
			if memoType.Valid {
				tx.MemoType = memoType.String
			}
			if memoValue.Valid {
				tx.MemoValue = memoValue.String
			}
			transactions = append(transactions, tx)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": transactions})
}

func (ic *IngesterController) GetTransaction(c *gin.Context) {
	hash := c.Param("hash")
	var tx models.Transaction
	var memoType, memoValue sql.NullString
	err := ic.db.QueryRow(`
		SELECT id, hash, ledger, index, source_account, fee_paid,
		       operation_count, created_at, memo_type, memo_value, successful
		FROM transactions WHERE hash = $1`, hash).Scan(
		&tx.ID, &tx.Hash, &tx.Ledger, &tx.Index, &tx.SourceAccount, &tx.FeePaid,
		&tx.OperationCount, &tx.CreatedAt, &memoType, &memoValue, &tx.Successful)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch transaction"})
		return
	}
	if memoType.Valid {
		tx.MemoType = memoType.String
	}
	if memoValue.Valid {
		tx.MemoValue = memoValue.String
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tx})
}

func (ic *IngesterController) GetOperations(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")

	rows, err := ic.db.Query(`
		SELECT id, transaction_id, index, type, source_account, details
		FROM operations
		ORDER BY id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch operations"})
		return
	}
	defer rows.Close()

	var operations []models.Operation
	for rows.Next() {
		var op models.Operation
		var sourceAccount sql.NullString
		if err := rows.Scan(&op.ID, &op.TransactionID, &op.Index,
			&op.Type, &sourceAccount, &op.Details); err == nil {
			if sourceAccount.Valid {
				op.SourceAccount = sourceAccount.String
			}
			operations = append(operations, op)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": operations})
}

func (ic *IngesterController) GetContractEvents(c *gin.Context) {
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")
	contractID := c.Query("contract_id")

	query := `
		SELECT id, contract_id, ledger, transaction_hash, event_type,
		       topics, data, in_successful_tx
		FROM contract_events`
	args := []interface{}{}
	if contractID != "" {
		query += " WHERE contract_id = $1"
		args = append(args, contractID)
	}
	query += " ORDER BY ledger DESC"
	if contractID != "" {
		query += " LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	} else {
		query += " LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}
	rows, err := ic.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch contract events"})
		return
	}
	defer rows.Close()

	var events []models.ContractEvent
	for rows.Next() {
		var event models.ContractEvent
		var topicsJSON, dataJSON []byte
		if err := rows.Scan(&event.ID, &event.ContractID, &event.Ledger,
			&event.TransactionHash, &event.EventType, &topicsJSON, &dataJSON, &event.InSuccessfulTx); err == nil {
			json.Unmarshal(topicsJSON, &event.Topics)
			event.Data = dataJSON
			events = append(events, event)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

func (ic *IngesterController) GetStats(c *gin.Context) {
	stats := *ic.stats
	ic.db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&stats.TransactionCount)
	ic.db.QueryRow("SELECT COUNT(*) FROM contract_events").Scan(&stats.EventCount)
	ic.db.QueryRow("SELECT COUNT(*) FROM operations").Scan(&stats.OperationCount)
	ic.db.QueryRow("SELECT COUNT(*) FROM ledgers").Scan(&stats.LedgersProcessed)
	stats.LastUpdateTime = time.Now()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}
