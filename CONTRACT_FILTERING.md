# Contract Filtering Configuration

The ingester supports filtering data to only include operations, transactions, and events for specific smart contract addresses.

## Configuration

### Config File (Recommended)
Contract filtering is configured in the environment-specific YAML files:

#### Testnet Configuration (`config/testnet.yaml`)
```yaml
stellar:
  filter_contracts:
    - "CADB73DZ7QP5BG5ZG6MRRL3J3X4WWHBCJ7PMCVZXYG7ZGCPIO2XCDBOM"
    - "CAD6YMZCO4Q3L5XZT2FD3MDHP3ZHFMYL24RZYG4YQAL4XQKVGVXYPSQQ"
```

#### Mainnet Configuration (`config/mainnet.yaml`)
```yaml
stellar:
  filter_contracts:
    - "YOUR_MAINNET_CONTRACT_ADDRESS_1"
    - "YOUR_MAINNET_CONTRACT_ADDRESS_2"
```

Note: Update the mainnet addresses with your actual mainnet contract addresses.

### Environment Variable (Override)
You can override the config file contracts by setting the `FILTER_CONTRACTS` environment variable:

```bash
export FILTER_CONTRACTS="CONTRACT_ADDRESS_1,CONTRACT_ADDRESS_2,CONTRACT_ADDRESS_3"
```

The environment variable takes precedence over the config file settings.

### Disable Filtering
To disable filtering and ingest all data:

1. Remove the `filter_contracts` section from your config file, OR
2. Set an empty environment variable:
```bash
export FILTER_CONTRACTS=""
```

## Environment Selection

The application uses different config files based on the environment:

```bash
# For testnet (uses config/testnet.yaml)
./sorobangraph -e development

# For mainnet (uses config/mainnet.yaml)  
./sorobangraph -e production

# For test environment (uses config/test.yaml)
./sorobangraph -e test
```

## What Gets Filtered

When contract filtering is enabled, the ingester will only store:

1. **Transactions** - Only transactions that contain at least one operation invoking one of the filtered contracts
2. **Operations** - Only `invoke_host_function` operations that target one of the filtered contracts  
3. **Contract Events** - Only events emitted by one of the filtered contracts

All other data (ledgers, accounts, etc. not related to these contracts) will be skipped to keep the database focused on your specific contracts.

## Verification

When the ingester starts, it will log which contracts it's filtering for:
```
INFO[0000] Ingester configured to filter for contracts: [CADB73DZ7QP5BG5ZG6MRRL3J3X4WWHBCJ7PMCVZXYG7ZGCPIO2XCDBOM CAD6YMZCO4Q3L5XZT2FD3MDHP3ZHFMYL24RZYG4YQAL4XQKVGVXYPSQQ]
```

Or if no filtering is configured:
```
INFO[0000] No contract filtering configured - ingesting all data
```