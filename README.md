# Soroban Graph Stellar Ingester

A Stellar blockchain data ingester that processes ledgers, transactions, and Soroban contract events into PostgreSQL.

## Project Structure

```
├── main.go                 # Application entry point
├── controllers/            # HTTP routing and request handling
│   ├── ingester.go        # Ingester API endpoints
│   └── user.go            # User controller (mock data)
├── handlers/               # Business logic implementation
│   └── ingester.go        # Stellar ingestion processing
├── models/                 # Data models split by entity
│   ├── stats.go           # Ingestion statistics
│   ├── transaction.go     # Transaction model
│   ├── operation.go       # Operation model
│   ├── event.go           # Contract event model
│   └── ledger.go          # Ledger model
├── db/                     # Database connection
├── server/                 # HTTP server setup
├── migrations/             # Database migrations
└── cmd/                    # Command line utilities
    ├── migrate/            # Database migration tool
    └── healthcheck/        # System health verification
```

## Setup ✅ COMPLETED

### 1. Database Setup

✅ **Database configured and tables created:**
```
postgresql://postgres:UikJouInuaAtzDYMdpsOlFxWPORldLBP@turntable.proxy.rlwy.net:52543/railway
```

### 2. Environment Configuration

✅ **`.env` file created with your database:**
```bash
DATABASE_URL=postgresql://postgres:UikJouInuaAtzDYMdpsOlFxWPORldLBP@turntable.proxy.rlwy.net:52543/railway
NETWORK_PASSPHRASE=Test SDF Network ; September 2015
HISTORY_ARCHIVE_URLS=https://history.stellar.org/prd/core-testnet/core_testnet_001
# ... other settings
```

### 3. Database Migration

✅ **Migrations completed successfully:**
```bash
go run cmd/migrate/main.go up
```

### 4. Test Setup

✅ **All tests passed:**
```bash
go run cmd/healthcheck/main.go
```

## Usage

### Build and Run

```bash
# Build the application
make build

# Run the application  
./sorobangraph-attest
```

## API Endpoints

### REST API

- `GET /health` - Health check
- `GET /api/v1/ledgers` - List ledgers
- `GET /api/v1/ledgers/:sequence` - Get specific ledger
- `GET /api/v1/transactions` - List transactions
- `GET /api/v1/transactions/:hash` - Get specific transaction
- `GET /api/v1/operations` - List operations
- `GET /api/v1/contract-events` - List Soroban events
- `GET /api/v1/stats` - Ingestion statistics

### WebSocket

Connect to `/api/v1/ws` for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message.type, message.data);
};
```

## Using Captive Core

For better performance, you can use a local Captive Core instance:

1. Install stellar-core:
   ```bash
   wget https://github.com/stellar/stellar-core/releases/download/v20.1.0/stellar-core_20.1.0_amd64.deb
   sudo dpkg -i stellar-core_20.1.0_amd64.deb
   ```

2. Set environment variables:
   ```bash
   export CAPTIVE_CORE_BINARY_PATH=/usr/bin/stellar-core
   export CAPTIVE_CORE_CONFIG_PATH=/etc/stellar-core.cfg
   ```

3. The ingester will automatically use Captive Core for ledger access

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `NETWORK_PASSPHRASE` | Stellar network to connect to | Testnet |
| `HISTORY_ARCHIVE_URLS` | History archive for ledger data | SDF Testnet |
| `CAPTIVE_CORE_BINARY_PATH` | Path to stellar-core binary | Optional |
| `CAPTIVE_CORE_CONFIG_PATH` | Path to stellar-core config | Optional |
| `START_LEDGER` | Ledger to start ingestion from | 0 (resume/genesis) |
| `END_LEDGER` | Ledger to stop at | 0 (continuous) |
| `PORT` | API server port | 8080 |
| `ENABLE_WEBSOCKET` | Enable WebSocket streaming | true |
| `LOG_LEVEL` | Logging verbosity | info |

## Database Schema

The ingester creates the following tables:

- `ledgers` - Ledger headers and metadata
- `transactions` - Transaction envelopes and results
- `operations` - Parsed operations with details
- `contract_events` - Soroban contract events
- `ingestion_state` - Tracks ingestion progress

## Performance Considerations

1. **Use Captive Core**: Local Captive Core provides faster ledger access
2. **Database Tuning**: Adjust PostgreSQL settings for write-heavy workloads
3. **Batch Processing**: The ingester processes entire ledgers atomically
4. **Indexing**: Proper indexes are created for common query patterns

## Monitoring

Check ingestion statistics:
```bash
curl http://localhost:8080/api/v1/stats
```

## Troubleshooting

1. **Slow ingestion**: Consider using local Captive Core
2. **Database connection issues**: Check PostgreSQL is running and accessible
3. **Missing ledgers**: The ingester will automatically retry failed ledgers
4. **WebSocket disconnections**: Clients should implement reconnection logic

## License

MIT