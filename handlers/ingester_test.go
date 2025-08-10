package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractFiltering(t *testing.T) {
	tests := []struct {
		name             string
		filterContracts  []string
		contractAddress  string
		expectedIncluded bool
	}{
		{
			name:             "No filter - should include all",
			filterContracts:  []string{},
			contractAddress:  "abc123",
			expectedIncluded: true,
		},
		{
			name:             "Contract in filter list",
			filterContracts:  []string{"abc123", "def456"},
			contractAddress:  "abc123",
			expectedIncluded: true,
		},
		{
			name:             "Contract not in filter list",
			filterContracts:  []string{"abc123", "def456"},
			contractAddress:  "xyz789",
			expectedIncluded: false,
		},
		{
			name:             "Empty contract address",
			filterContracts:  []string{"abc123"},
			contractAddress:  "",
			expectedIncluded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.NewEntry(logrus.New())
			config := &Config{
				FilterContracts: tt.filterContracts,
			}

			ingester := &Ingester{
				config: config,
				logger: logger,
			}

			result := ingester.isFilteredContract(tt.contractAddress)
			assert.Equal(t, tt.expectedIncluded, result)
		})
	}
}

func TestIngesterInitialization(t *testing.T) {
	tests := []struct {
		name            string
		config          *Config
		expectWebSocket bool
		expectError     bool
	}{
		{
			name: "Basic initialization without WebSocket",
			config: &Config{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				StartLedger:       1000,
				EndLedger:         2000,
				EnableWebSocket:   false,
				LogLevel:          "info",
			},
			expectWebSocket: false,
			expectError:     false,
		},
		{
			name: "Initialization with WebSocket enabled",
			config: &Config{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				StartLedger:       1000,
				EndLedger:         0, // Continuous streaming
				EnableWebSocket:   true,
				LogLevel:          "debug",
			},
			expectWebSocket: true,
			expectError:     false,
		},
		{
			name: "Initialization with contract filters",
			config: &Config{
				NetworkPassphrase: "Test SDF Network ; September 2015",
				StartLedger:       1000,
				EndLedger:         5000,
				FilterContracts:   []string{"contract1", "contract2", "contract3"},
				EnableWebSocket:   false,
				LogLevel:          "warn",
			},
			expectWebSocket: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.NewEntry(logrus.New())
			ingester, err := NewIngester(tt.config, nil, logger)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, ingester)
			assert.Equal(t, tt.config, ingester.config)
			assert.Equal(t, tt.config.NetworkPassphrase, ingester.networkPassphrase)

			if tt.expectWebSocket {
				assert.NotNil(t, ingester.wsHub)
				assert.NotNil(t, ingester.wsHub.clients)
				assert.NotNil(t, ingester.wsHub.broadcast)
				assert.NotNil(t, ingester.wsHub.register)
				assert.NotNil(t, ingester.wsHub.unregister)
			} else {
				assert.Nil(t, ingester.wsHub)
			}

			// Check stats initialization
			assert.NotNil(t, ingester.stats)
			assert.NotZero(t, ingester.stats.StartTime)
		})
	}
}

func TestIngesterStats(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	// Test initial stats
	stats := ingester.Stats()
	assert.NotNil(t, stats)
	assert.Equal(t, uint32(0), stats.CurrentLedger)
	assert.Equal(t, int64(0), stats.TransactionCount)
	assert.Equal(t, int64(0), stats.OperationCount)
	assert.Equal(t, int64(0), stats.EventCount)

	// Test stats updates
	ingester.setCurrentLedger(1000)
	assert.Equal(t, uint32(1000), ingester.getCurrentLedger())
	assert.Equal(t, uint32(1000), ingester.stats.CurrentLedger)

	ingester.incrementTransactionCount()
	assert.Equal(t, int64(1), ingester.stats.TransactionCount)

	ingester.incrementOperationCount(5)
	assert.Equal(t, int64(5), ingester.stats.OperationCount)

	ingester.incrementEventCount()
	assert.Equal(t, int64(1), ingester.stats.EventCount)

	ingester.incrementLedgersProcessed()
	assert.Equal(t, int64(1), ingester.stats.LedgersProcessed)
	assert.Greater(t, ingester.stats.ProcessingRate, float64(0))
}

func TestIngesterStart(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("Start without ledger backend", func(t *testing.T) {
		config := &Config{
			NetworkPassphrase: "Test SDF Network ; September 2015",
			StartLedger:       1000,
			EndLedger:         2000,
		}

		ingester, err := NewIngester(config, nil, logger)
		require.NoError(t, err)

		ctx := context.Background()
		err = ingester.Start(ctx)
		assert.NoError(t, err) // Should not error when ledger backend is nil
	})

	t.Run("Start with WebSocket hub", func(t *testing.T) {
		config := &Config{
			NetworkPassphrase: "Test SDF Network ; September 2015",
			StartLedger:       1000,
			EnableWebSocket:   true,
		}

		ingester, err := NewIngester(config, nil, logger)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = ingester.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, ingester.wsHub)
	})
}

func TestLedgerHelpers(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, nil, logger)
	require.NoError(t, err)

	// Test concurrent access to ledger state
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := uint32(1); i <= 100; i++ {
			ingester.setCurrentLedger(i)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = ingester.getCurrentLedger()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Final ledger should be 100
	assert.Equal(t, uint32(100), ingester.getCurrentLedger())
}

func TestDatabaseStateManagement(t *testing.T) {
	// Note: This test requires a real database connection
	// In a real scenario, you'd use a test database or mock
	t.Skip("Skipping database test - requires database connection")

	db, err := sql.Open("postgres", "postgresql://test:test@localhost/test?sslmode=disable")
	if err != nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	logger := logrus.NewEntry(logrus.New())
	config := &Config{
		NetworkPassphrase: "Test SDF Network ; September 2015",
	}

	ingester, err := NewIngester(config, db, logger)
	require.NoError(t, err)

	// Test loading last ledger
	lastLedger, err := ingester.loadLastLedger()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, lastLedger, uint32(0))
}
