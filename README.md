# Stellar Blockchain Ingester

A high-performance Go-based ingester for the Stellar blockchain using the native Stellar ingest package. This ingester provides real-time streaming of ledgers, transactions, operations, and Soroban contract events.

## Features

- **Native Stellar Ingest SDK**: Uses Stellar's official ingest package for direct ledger access
- **Multiple Backend Support**: 
  - Remote Captive Core (via History Archives)
  - Local Captive Core for maximum performance
- **Real-time WebSocket Streaming**: Live updates for transactions and events
- **Comprehensive Data Ingestion**:
  - Ledgers with full metadata
  - Transactions with XDR storage
  - Operations with detailed parsing
  - Soroban contract events
- **RESTful API**: Query ingested data via HTTP endpoints
- **PostgreSQL Storage**: Efficient querying with proper indexing
- **Resume Capability**: Automatically resumes from last processed ledger

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository
2. Copy the environment file:
   ```bash
   cp .env.example .env
   ```

3. Start the services:
   ```bash
   docker-compose up -d
   ```

4. Check logs:
   ```bash
   docker-compose logs -f ingester
   ```

### Local Development

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Set up PostgreSQL:
   ```bash
   createdb stellar_ingester
   ```

3. Set environment variables:
   ```bash
   export DATABASE_URL="postgres://user:password@localhost/stellar_ingester?sslmode=disable"
   export NETWORK_PASSPHRASE="Test SDF Network ; September 2015"
   export HISTORY_ARCHIVE_URLS="https://history.stellar.org/prd/core-testnet/core_testnet_001"
   ```

4. Run the ingester:
   ```bash
   go run main.go
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