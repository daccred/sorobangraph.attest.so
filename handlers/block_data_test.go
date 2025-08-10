package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/daccred/sorobangraph.attest.so/models"
	"github.com/sirupsen/logrus"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScValConversion(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	t.Run("ScVal to String conversions", func(t *testing.T) {
		tests := []struct {
			name     string
			scVal    xdr.ScVal
			expected string
		}{
			{
				name:     "Boolean true",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &[]bool{true}[0]},
				expected: "true",
			},
			{
				name:     "Boolean false",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &[]bool{false}[0]},
				expected: "false",
			},
			{
				name:     "Int32",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{42}[0]},
				expected: "42",
			},
			{
				name:     "Int64",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvI64, I64: &[]xdr.Int64{9999999}[0]},
				expected: "9999999",
			},
			{
				name:     "UInt32",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &[]xdr.Uint32{100}[0]},
				expected: "100",
			},
			{
				name:     "UInt64",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: &[]xdr.Uint64{1000000}[0]},
				expected: "1000000",
			},
			{
				name:     "Symbol",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &[]xdr.ScSymbol{xdr.ScSymbol("test_symbol")}[0]},
				expected: "test_symbol",
			},
			{
				name:     "String",
				scVal:    xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &[]xdr.ScString{xdr.ScString("hello world")}[0]},
				expected: "hello world",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ingester.scValToString(tt.scVal)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("ScVal to JSON conversions", func(t *testing.T) {
		// Test boolean
		boolVal := xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &[]bool{true}[0]}
		result := ingester.scValToJSON(boolVal)
		assert.Equal(t, true, result)

		// Test integer
		intVal := xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{42}[0]}
		result = ingester.scValToJSON(intVal)
		assert.Equal(t, xdr.Int32(42), result)

		// Test string
		strVal := xdr.ScVal{Type: xdr.ScValTypeScvString, Str: &[]xdr.ScString{xdr.ScString("test")}[0]}
		result = ingester.scValToJSON(strVal)
		assert.Equal(t, "test", result)

		// Test vector
		vec := &xdr.ScVec{
			xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{1}[0]},
			xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{2}[0]},
			xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{3}[0]},
		}
		vecVal := xdr.ScVal{Type: xdr.ScValTypeScvVec, Vec: &vec}
		result = ingester.scValToJSON(vecVal)
		resultArray, ok := result.([]interface{})
		assert.True(t, ok)
		assert.Len(t, resultArray, 3)

		// Test map
		mapEntries := &xdr.ScMap{
			{
				Key: xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: &[]xdr.ScSymbol{xdr.ScSymbol("key1")}[0]},
				Val: xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: &[]xdr.Int32{100}[0]},
			},
		}
		mapVal := xdr.ScVal{Type: xdr.ScValTypeScvMap, Map: &mapEntries}
		result = ingester.scValToJSON(mapVal)
		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, resultMap, "key1")
	})
}

func TestLedgerInfoProcessing(t *testing.T) {
	ledgerInfo := models.LedgerInfo{
		Sequence:         1000,
		Hash:             "abc123def456",
		PreviousHash:     "xyz789",
		TransactionCount: 10,
		OperationCount:   25,
		ClosedAt:         time.Now(),
		TotalCoins:       1000000000,
		FeePool:          500000,
		BaseFee:          100,
		BaseReserve:      10000000,
		MaxTxSetSize:     1000,
		ProtocolVersion:  20,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(ledgerInfo)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled models.LedgerInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, ledgerInfo.Sequence, unmarshaled.Sequence)
	assert.Equal(t, ledgerInfo.Hash, unmarshaled.Hash)
	assert.Equal(t, ledgerInfo.TransactionCount, unmarshaled.TransactionCount)
	assert.Equal(t, ledgerInfo.OperationCount, unmarshaled.OperationCount)
}

func TestExtractContractAddress(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	t.Run("Contract address extraction", func(t *testing.T) {
		// Create a contract ID
		contractHash := xdr.Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
		contractAddress := xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: &contractHash,
		}

		invokeArgs := xdr.InvokeContractArgs{
			ContractAddress: contractAddress,
			FunctionName:    xdr.ScSymbol("test_function"),
			Args:            xdr.ScVec{},
		}

		result := ingester.extractContractAddress(invokeArgs)
		expected := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
		assert.Equal(t, expected, result)
	})

	t.Run("Non-contract address returns empty", func(t *testing.T) {
		// Create an account address instead of contract
		accountId := xdr.AccountId{
			Type: xdr.PublicKeyTypePublicKeyTypeEd25519,
		}
		accountAddress := xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &accountId,
		}

		invokeArgs := xdr.InvokeContractArgs{
			ContractAddress: accountAddress,
			FunctionName:    xdr.ScSymbol("test_function"),
			Args:            xdr.ScVec{},
		}

		result := ingester.extractContractAddress(invokeArgs)
		assert.Equal(t, "", result)
	})
}

func TestWebSocketHub(t *testing.T) {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}

	// Test client registration
	client := &WebSocketClient{
		send: make(chan interface{}, 256),
		hub:  hub,
	}

	// Start hub in background
	go hub.run()

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // Give time for registration

	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()
	assert.True(t, exists, "Client should be registered")

	// Test broadcast
	testMessage := map[string]interface{}{
		"type": "test",
		"data": "test_data",
	}
	hub.broadcast <- testMessage

	select {
	case msg := <-client.send:
		msgMap, ok := msg.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test", msgMap["type"])
		assert.Equal(t, "test_data", msgMap["data"])
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive broadcast message")
	}

	// Unregister client
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond) // Give time for unregistration

	hub.mu.RLock()
	_, exists = hub.clients[client]
	hub.mu.RUnlock()
	assert.False(t, exists, "Client should be unregistered")
}

func TestStatsUpdateAndTracking(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	// Test concurrent stats updates
	done := make(chan bool)
	numGoroutines := 10
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < incrementsPerGoroutine; j++ {
				ingester.incrementTransactionCount()
				ingester.incrementOperationCount(1)
				ingester.incrementEventCount()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	expectedCount := int64(numGoroutines * incrementsPerGoroutine)
	assert.Equal(t, expectedCount, ingester.stats.TransactionCount)
	assert.Equal(t, expectedCount, ingester.stats.OperationCount)
	assert.Equal(t, expectedCount, ingester.stats.EventCount)
}

func TestProcessingRateCalculation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	// Set a start time in the past
	ingester.stats.StartTime = time.Now().Add(-10 * time.Second)

	// Increment ledgers processed
	for i := 0; i < 100; i++ {
		ingester.incrementLedgersProcessed()
	}

	assert.Equal(t, int64(100), ingester.stats.LedgersProcessed)
	assert.Greater(t, ingester.stats.ProcessingRate, float64(0))

	// Processing rate should be approximately 10 ledgers/second
	// (100 ledgers in ~10 seconds)
	assert.InDelta(t, 10.0, ingester.stats.ProcessingRate, 2.0)
}
