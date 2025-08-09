package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/ingest"
	backends "github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/daccred/sorobangraph.attest.so/models"
)

// Ingester handles the data ingestion from Stellar
type Ingester struct {
	config            *Config
	db                *sql.DB
	ledgerBackend     backends.LedgerBackend
	networkPassphrase string
	wsHub             *WebSocketHub
	mu                sync.RWMutex
	stats             *models.Stats
	currentLedger     uint32
	logger            *logrus.Entry
}

// Config holds the ingestion configuration
type Config struct {
	NetworkPassphrase     string
	CaptiveCoreConfigPath string
	CaptiveCoreBinaryPath string
	HistoryArchiveURLs    []string
	StartLedger           uint32
	EndLedger             uint32 // 0 means continuous streaming
	EnableWebSocket       bool
	LogLevel              string
}

// WebSocket structures
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan interface{}
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

type WebSocketClient struct {
	send chan interface{}
	hub  *WebSocketHub
}

func NewIngester(cfg *Config, db *sql.DB, logger *logrus.Entry) (*Ingester, error) {
	// Setup logging level
	log.SetLevel(logrus.InfoLevel)
	if cfg.LogLevel != "" {
		if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
			log.SetLevel(level)
		}
	}

	// For now, we'll disable the ledger backend initialization for testing
	// In production, you'll need to configure Stellar Core properly
	var ledgerBackend backends.LedgerBackend = nil

	ingester := &Ingester{
		config:            cfg,
		db:                db,
		ledgerBackend:     ledgerBackend,
		networkPassphrase: cfg.NetworkPassphrase,
		logger:            logger,
		stats: &models.Stats{StartTime: time.Now()},
	}

	if cfg.EnableWebSocket {
		ingester.wsHub = &WebSocketHub{
			clients:    make(map[*WebSocketClient]bool),
			broadcast:  make(chan interface{}, 256),
			register:   make(chan *WebSocketClient),
			unregister: make(chan *WebSocketClient),
		}
	}

	return ingester, nil
}

func (i *Ingester) Stats() *models.Stats { return i.stats }

// Start begins the ingestion process using Stellar's ingest package
func (i *Ingester) Start(ctx context.Context) error {
	// Load last ingestion state
	startLedger := i.config.StartLedger
	if lastLedger, err := i.loadLastLedger(); err == nil && lastLedger > 0 {
		startLedger = lastLedger + 1
		i.logger.Infof("Resuming from ledger %d", startLedger)
	}

	if i.config.EnableWebSocket && i.wsHub != nil {
		go i.wsHub.run()
	}
	go i.updateStats(ctx)

	var ledgerRange backends.Range
	if i.config.EndLedger > 0 {
		ledgerRange = backends.BoundedRange(startLedger, i.config.EndLedger)
	} else {
		ledgerRange = backends.UnboundedRange(startLedger)
	}

	i.logger.Infof("Starting ingestion from ledger %d", startLedger)

	// If no ledger backend is configured, skip ingestion gracefully
	if i.ledgerBackend == nil {
		i.logger.Warn("Ledger backend not configured; skipping ingestion")
		return nil
	}

	if err := i.ledgerBackend.PrepareRange(ctx, ledgerRange); err != nil {
		return fmt.Errorf("failed to prepare range: %w", err)
	}
	go i.processLedgers(ctx)
	return nil
}

func (i *Ingester) processLedgers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Context cancelled, stopping ledger processing")
			return
		default:
			lcm, err := i.ledgerBackend.GetLedger(ctx, i.getCurrentLedger()+1)
			if err != nil {
				if err == io.EOF {
					time.Sleep(2 * time.Second)
					continue
				}
				i.logger.Errorf("Failed to get ledger: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			if err := i.processLedger(lcm); err != nil {
				i.logger.Errorf("Failed to process ledger %d: %v", lcm.LedgerSequence(), err)
				continue
			}
			i.setCurrentLedger(lcm.LedgerSequence())
			i.incrementLedgersProcessed()
			i.logger.Infof("Processed ledger %d with %d transactions", lcm.LedgerSequence(), lcm.CountTransactions())
		}
	}
}

func (i *Ingester) processLedger(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	ledgerSeq := ledgerCloseMeta.LedgerSequence()
	ledgerHeader := ledgerCloseMeta.LedgerHeaderHistoryEntry()

	changeReader, err := ingest.NewLedgerChangeReaderFromLedgerCloseMeta(i.networkPassphrase, ledgerCloseMeta)
	if err != nil {
		return fmt.Errorf("failed to create change reader: %w", err)
	}
	defer changeReader.Close()

	dbTx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Count operations in all transactions
	operationCount := 0
	txs := ledgerCloseMeta.TransactionEnvelopes()
	for _, tx := range txs {
		operationCount += len(tx.Operations())
	}

	ledgerInfo := models.LedgerInfo{
		Sequence:         ledgerSeq,
		Hash:             fmt.Sprintf("%x", ledgerHeader.Hash),
		PreviousHash:     fmt.Sprintf("%x", ledgerHeader.Header.PreviousLedgerHash),
		TransactionCount: ledgerCloseMeta.CountTransactions(),
		OperationCount:   operationCount,
		ClosedAt:         time.Unix(int64(ledgerHeader.Header.ScpValue.CloseTime), 0),
		TotalCoins:       int64(ledgerHeader.Header.TotalCoins),
		FeePool:          int64(ledgerHeader.Header.FeePool),
		BaseFee:          uint32(ledgerHeader.Header.BaseFee),
		BaseReserve:      uint32(ledgerHeader.Header.BaseReserve),
		MaxTxSetSize:     uint32(ledgerHeader.Header.MaxTxSetSize),
		ProtocolVersion:  uint32(ledgerHeader.Header.LedgerVersion),
	}
	if err := i.storeLedger(dbTx, ledgerInfo); err != nil {
		return fmt.Errorf("failed to store ledger: %w", err)
	}

	txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(i.networkPassphrase, ledgerCloseMeta)
	if err != nil {
		return fmt.Errorf("failed to create transaction reader: %w", err)
	}
	defer txReader.Close()

	for {
		tx, err := txReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read transaction: %w", err)
		}
		if err := i.processTransaction(dbTx, ledgerSeq, tx); err != nil {
			i.logger.Errorf("Failed to process transaction in ledger %d: %v", ledgerSeq, err)
		}
	}

	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read change: %w", err)
		}
		switch change.Type {
		case xdr.LedgerEntryTypeAccount:
			// TODO: handle account changes
		case xdr.LedgerEntryTypeData:
			// TODO: handle data entries
		case xdr.LedgerEntryTypeContractData:
			// TODO: handle Soroban contract data
		case xdr.LedgerEntryTypeContractCode:
			// TODO: handle Soroban contract code
		}
	}

	if err := i.updateIngestionState(dbTx, ledgerSeq); err != nil {
		return fmt.Errorf("failed to update ingestion state: %w", err)
	}
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if i.wsHub != nil {
		i.wsHub.broadcast <- map[string]interface{}{"type": "ledger", "data": ledgerInfo}
	}
	return nil
}

func (i *Ingester) processTransaction(dbTx *sql.Tx, ledgerSeq uint32, tx ingest.LedgerTransaction) error {
	txHash := tx.Result.TransactionHash.HexString()
	envelope := tx.Envelope
	sourceAccount := envelope.SourceAccount().ToAccountId().Address()
	successful := tx.Result.Successful()

	feePaid := int64(envelope.Fee())

	var memoType, memoValue string
	memo := envelope.Memo()
	switch memo.Type {
	case xdr.MemoTypeMemoText:
		memoType = "text"
		memoValue = string(memo.MustText())
	case xdr.MemoTypeMemoId:
		memoType = "id"
		memoValue = fmt.Sprintf("%d", memo.MustId())
	case xdr.MemoTypeMemoHash:
		memoType = "hash"
		memoValue = fmt.Sprintf("%x", memo.MustHash())
	case xdr.MemoTypeMemoReturn:
		memoType = "return"
		memoValue = fmt.Sprintf("%x", memo.MustRetHash())
	}

	transaction := models.Transaction{
		ID:             fmt.Sprintf("%d-%d", ledgerSeq, tx.Index),
		Hash:           txHash,
		Ledger:         ledgerSeq,
		Index:          tx.Index,
		SourceAccount:  sourceAccount,
		FeePaid:        feePaid,
		OperationCount: int32(len(envelope.Operations())),
		CreatedAt:      time.Now(), // Use current time as LedgerCloseTime is not directly available
		MemoType:       memoType,
		MemoValue:      memoValue,
		Successful:     successful,
	}

	envelopeXDR, _ := envelope.MarshalBinary()
	resultXDR, _ := tx.Result.MarshalBinary()
	metaXDR, _ := tx.UnsafeMeta.MarshalBinary()

	if _, err := dbTx.Exec(`
		INSERT INTO transactions (id, hash, ledger, index, source_account, fee_paid,
			operation_count, created_at, memo_type, memo_value, successful,
			envelope_xdr, result_xdr, result_meta_xdr)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO NOTHING`,
		transaction.ID, transaction.Hash, transaction.Ledger, transaction.Index,
		transaction.SourceAccount, transaction.FeePaid, transaction.OperationCount,
		transaction.CreatedAt, transaction.MemoType, transaction.MemoValue,
		transaction.Successful, envelopeXDR, resultXDR, metaXDR); err != nil {
		return fmt.Errorf("failed to store transaction: %w", err)
	}

	operations := envelope.Operations()
	for opIndex, op := range operations {
		if err := i.processOperation(dbTx, transaction.ID, uint32(opIndex), op, tx); err != nil {
			i.logger.Errorf("Failed to process operation %d in tx %s: %v", opIndex, txHash, err)
		}
	}

	if tx.UnsafeMeta.V == 3 && tx.UnsafeMeta.V3 != nil {
		if err := i.processSorobanEvents(dbTx, tx); err != nil {
			i.logger.Errorf("Failed to process Soroban events in tx %s: %v", txHash, err)
		}
	}
	i.incrementTransactionCount()
	if i.wsHub != nil {
		i.wsHub.broadcast <- map[string]interface{}{"type": "transaction", "data": transaction}
	}
	return nil
}

func (i *Ingester) processOperation(dbTx *sql.Tx, txID string, index uint32, op xdr.Operation, tx ingest.LedgerTransaction) error {
	opID := fmt.Sprintf("%s-%d", txID, index)
	var sourceAccount string
	if op.SourceAccount != nil {
		sourceAccount = op.SourceAccount.ToAccountId().Address()
	}
	var opType string
	var details map[string]interface{}
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		opType = "create_account"
		createOp := op.Body.MustCreateAccountOp()
		details = map[string]interface{}{"destination": createOp.Destination.Address(), "starting_balance": createOp.StartingBalance}
	case xdr.OperationTypePayment:
		opType = "payment"
		paymentOp := op.Body.MustPaymentOp()
		details = map[string]interface{}{"destination": paymentOp.Destination.ToAccountId().Address(), "amount": paymentOp.Amount}
	case xdr.OperationTypeManageSellOffer:
		opType = "manage_sell_offer"
		offerOp := op.Body.MustManageSellOfferOp()
		details = map[string]interface{}{"amount": offerOp.Amount}
	case xdr.OperationTypeCreatePassiveSellOffer:
		opType = "create_passive_sell_offer"
		passiveOp := op.Body.MustCreatePassiveSellOfferOp()
		details = map[string]interface{}{"amount": passiveOp.Amount}
	case xdr.OperationTypeSetOptions:
		opType = "set_options"
		details = map[string]interface{}{}
	case xdr.OperationTypeChangeTrust:
		opType = "change_trust"
		details = map[string]interface{}{}
	case xdr.OperationTypeAllowTrust:
		opType = "allow_trust"
		details = map[string]interface{}{}
	case xdr.OperationTypeAccountMerge:
		opType = "account_merge"
		details = map[string]interface{}{}
	case xdr.OperationTypeManageData:
		opType = "manage_data"
		details = map[string]interface{}{}
	case xdr.OperationTypeInvokeHostFunction:
		opType = "invoke_host_function"
		details = map[string]interface{}{"function_type": op.Body.MustInvokeHostFunctionOp().HostFunction.Type.String()}
	case xdr.OperationTypeExtendFootprintTtl:
		opType = "extend_footprint_ttl"
		details = map[string]interface{}{"extend_to": op.Body.MustExtendFootprintTtlOp().ExtendTo}
	case xdr.OperationTypeRestoreFootprint:
		opType = "restore_footprint"
		details = map[string]interface{}{}
	default:
		opType = op.Body.Type.String()
		details = map[string]interface{}{}
	}
	detailsJSON, _ := json.Marshal(details)
	if _, err := dbTx.Exec(`
		INSERT INTO operations (id, transaction_id, index, type, source_account, details)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`, opID, txID, index, opType, sourceAccount, detailsJSON); err != nil {
		return fmt.Errorf("failed to store operation: %w", err)
	}
	i.incrementOperationCount(1)
	return nil
}

func (i *Ingester) processSorobanEvents(dbTx *sql.Tx, tx ingest.LedgerTransaction) error {
	if tx.UnsafeMeta.V != 3 || tx.UnsafeMeta.V3 == nil {
		return nil
	}
	txHash := tx.Result.TransactionHash.HexString()
	successful := tx.Result.Successful()
	// Process Soroban events from meta if transaction was successful
	if successful && tx.UnsafeMeta.V3.SorobanMeta != nil {
		for _, event := range tx.UnsafeMeta.V3.SorobanMeta.Events {
			// Get ledger from current ingester state
			ledger := i.getCurrentLedger()
			if err := i.storeSorobanEvent(dbTx, event, ledger, txHash, true); err != nil {
				i.logger.Errorf("Failed to store Soroban event: %v", err)
			}
		}
	}
	return nil
}

func (i *Ingester) storeSorobanEvent(dbTx *sql.Tx, event xdr.ContractEvent, ledger uint32, txHash string, successful bool) error {
	var contractID string
	if event.ContractId != nil {
		contractID = fmt.Sprintf("%x", *event.ContractId)
	}
	eventType := "unknown"
	if event.Type == xdr.ContractEventTypeContract {
		eventType = "contract"
	} else if event.Type == xdr.ContractEventTypeSystem {
		eventType = "system"
	}
	var topics []string
	for _, topic := range event.Body.V0.Topics {
		topics = append(topics, i.scValToString(topic))
	}
	data := i.scValToJSON(event.Body.V0.Data)
	eventID := fmt.Sprintf("%s-%d-%s", txHash, len(topics), contractID)
	topicsJSON, _ := json.Marshal(topics)
	dataJSON, _ := json.Marshal(data)
	if _, err := dbTx.Exec(`
		INSERT INTO contract_events (id, contract_id, ledger, transaction_hash,
			event_type, topics, data, in_successful_tx)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING`, eventID, contractID, ledger, txHash, eventType, topicsJSON, dataJSON, successful); err != nil {
		return fmt.Errorf("failed to store contract event: %w", err)
	}
	i.incrementEventCount()
	if i.wsHub != nil {
		i.wsHub.broadcast <- map[string]interface{}{"type": "contract_event", "data": models.ContractEvent{ID: eventID, ContractID: contractID, Ledger: ledger, TransactionHash: txHash, EventType: eventType, Topics: topics, Data: dataJSON, InSuccessfulTx: successful}}
	}
	return nil
}

// Helpers
func (i *Ingester) getCurrentLedger() uint32 { i.mu.RLock(); defer i.mu.RUnlock(); return i.currentLedger }
func (i *Ingester) setCurrentLedger(ledger uint32) { i.mu.Lock(); defer i.mu.Unlock(); i.currentLedger = ledger; i.stats.CurrentLedger = ledger }
func (i *Ingester) incrementTransactionCount() { i.mu.Lock(); defer i.mu.Unlock(); i.stats.TransactionCount++ }
func (i *Ingester) incrementOperationCount(count int64) { i.mu.Lock(); defer i.mu.Unlock(); i.stats.OperationCount += count }
func (i *Ingester) incrementEventCount() { i.mu.Lock(); defer i.mu.Unlock(); i.stats.EventCount++ }
func (i *Ingester) incrementLedgersProcessed() { i.mu.Lock(); defer i.mu.Unlock(); i.stats.LedgersProcessed++; elapsed := time.Since(i.stats.StartTime).Seconds(); if elapsed > 0 { i.stats.ProcessingRate = float64(i.stats.LedgersProcessed) / elapsed } }
func (i *Ingester) updateStats(ctx context.Context) { ticker := time.NewTicker(30 * time.Second); defer ticker.Stop(); for { select { case <-ctx.Done(): return; case <-ticker.C: i.mu.Lock(); i.stats.LastUpdateTime = time.Now(); i.mu.Unlock() } } }

// XDR helpers
func (i *Ingester) scValToString(val xdr.ScVal) string {
	switch val.Type {
	case xdr.ScValTypeScvBool:
		return fmt.Sprintf("%v", val.MustB())
	case xdr.ScValTypeScvI32:
		return fmt.Sprintf("%d", val.MustI32())
	case xdr.ScValTypeScvI64:
		return fmt.Sprintf("%d", val.MustI64())
	case xdr.ScValTypeScvU32:
		return fmt.Sprintf("%d", val.MustU32())
	case xdr.ScValTypeScvU64:
		return fmt.Sprintf("%d", val.MustU64())
	case xdr.ScValTypeScvSymbol:
		return string(val.MustSym())
	case xdr.ScValTypeScvString:
		return string(val.MustStr())
	case xdr.ScValTypeScvBytes:
		return fmt.Sprintf("%x", val.MustBytes())
	default:
		data, _ := val.MarshalBinary()
		return fmt.Sprintf("%x", data)
	}
}

func (i *Ingester) scValToJSON(val xdr.ScVal) interface{} {
	switch val.Type {
	case xdr.ScValTypeScvBool:
		return val.MustB()
	case xdr.ScValTypeScvI32:
		return val.MustI32()
	case xdr.ScValTypeScvI64:
		return val.MustI64()
	case xdr.ScValTypeScvU32:
		return val.MustU32()
	case xdr.ScValTypeScvU64:
		return val.MustU64()
	case xdr.ScValTypeScvSymbol:
		return string(val.MustSym())
	case xdr.ScValTypeScvString:
		return string(val.MustStr())
	case xdr.ScValTypeScvBytes:
		return fmt.Sprintf("%x", val.MustBytes())
	case xdr.ScValTypeScvVec:
		vec := val.MustVec()
		result := make([]interface{}, len(*vec))
		for idx, item := range *vec {
			result[idx] = i.scValToJSON(item)
		}
		return result
	case xdr.ScValTypeScvMap:
		m := val.MustMap()
		result := make(map[string]interface{})
		for _, entry := range *m {
			key := i.scValToString(entry.Key)
			result[key] = i.scValToJSON(entry.Val)
		}
		return result
	default:
		data, _ := val.MarshalBinary()
		return fmt.Sprintf("%x", data)
	}
}

// DB helpers
func (i *Ingester) storeLedger(tx *sql.Tx, ledger models.LedgerInfo) error {
	ledgerHeaderJSON, _ := json.Marshal(ledger)
	_, err := tx.Exec(`
		INSERT INTO ledgers (sequence, hash, previous_hash, transaction_count,
			operation_count, closed_at, total_coins, fee_pool, base_fee,
			base_reserve, max_tx_set_size, protocol_version, ledger_header)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (sequence) DO NOTHING`,
		ledger.Sequence, ledger.Hash, ledger.PreviousHash, ledger.TransactionCount,
		ledger.OperationCount, ledger.ClosedAt, ledger.TotalCoins, ledger.FeePool,
		ledger.BaseFee, ledger.BaseReserve, ledger.MaxTxSetSize, ledger.ProtocolVersion, ledgerHeaderJSON)
	return err
}

func (i *Ingester) updateIngestionState(tx *sql.Tx, ledger uint32) error {
	_, err := tx.Exec(`
		INSERT INTO ingestion_state (id, last_ledger, updated_at)
		VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
			last_ledger = EXCLUDED.last_ledger,
			updated_at = EXCLUDED.updated_at`, ledger, time.Now())
	return err
}

func (i *Ingester) loadLastLedger() (uint32, error) {
	var lastLedger uint32
	err := i.db.QueryRow(`SELECT last_ledger FROM ingestion_state WHERE id = 1`).Scan(&lastLedger)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return lastLedger, err
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock(); h.clients[client] = true; h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock(); if _, ok := h.clients[client]; ok { delete(h.clients, client); close(client.send) }; h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select { case client.send <- message: default: delete(h.clients, client); close(client.send) }
			}
			h.mu.RUnlock()
		}
	}
}