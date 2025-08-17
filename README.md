# ProgPoW WASM Implementation

A standalone Go module for compiling ProgPoW (Programmatic Proof-of-Work) verification to WebAssembly, enabling browser-based and Node.js block verification for the Quai network.

## 🏗️ Project Structure

```
.
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── progpow/                  # Core ProgPoW algorithm implementation
│   ├── algorithm_progpow_wasm.go  # ProgPoW algorithm
│   ├── algorithm_wasm.go          # Ethash cache generation
│   └── bitutil.go                 # Bitwise utilities
├── cmd/                      # WASM entry point
│   ├── main.go              # Main WASM exports
│   ├── seal_hash.go         # Seal hash computation
│   ├── proto_defs.proto     # Protobuf definitions
│   └── proto_defs.pb.go     # Generated protobuf code
├── progpow_full.wasm        # Compiled WASM module
├── build.sh                 # Build script
├── serve.sh                 # HTTP server for testing
├── wasm_exec.js             # Go WASM runtime
├── test_progpow_updated.html # Browser-based verification UI
├── verify_block.js          # Node.js CLI verification tool
├── package.json             # Node.js dependencies
└── package-lock.json        # Dependency lock file
```

## 📦 Installation

### Prerequisites
- Go 1.21 or later
- Node.js 16+ (for CLI testing)
- Protocol Buffers compiler (optional, for regenerating protobuf code)

### Install Node Dependencies
```bash
npm install
```

## 🔨 Building

### Quick Build
```bash
./build.sh
```

### Manual Build
```bash
cd cmd
GOOS=js GOARCH=wasm go build -o ../progpow_full.wasm .
```

### Regenerate Protobuf (if needed)
```bash
cd cmd
protoc --go_out=. --go_opt=paths=source_relative proto_defs.proto
```

## 🧪 Testing

### Browser Testing
1. Start the HTTP server:
   ```bash
   ./serve.sh
   ```

2. Open in browser:
   ```
   http://localhost:8080/test_progpow_updated.html
   ```

3. Features:
   - Fetch and verify blocks from Quai network RPC
   - Manual block data input
   - Real-time verification with detailed results

### Node.js CLI Testing
```bash
# Verify latest block
node verify_block.js

# Verify specific block
node verify_block.js 1000

# Verify block range
node verify_block.js --range 1 10

# Use custom RPC endpoint
export QUAI_RPC=https://rpc.quai.network/cyprus1
node verify_block.js
```

## 📚 API Functions

The compiled WASM module exports the following functions to JavaScript:

### `computeProgPoW(headerHash, nonce, blockNumber, primeTerminusNumber)`
Computes the ProgPoW mix hash and pow hash.

**Parameters:**
- `headerHash` - 32-byte hex string (seal hash)
- `nonce` - Hex string or number
- `blockNumber` - Block number as integer
- `primeTerminusNumber` - Prime terminus number as integer

**Returns:**
```javascript
{
  mixHash: "0x...",     // 32-byte hex
  powHash: "0x...",     // 32-byte hex
  epoch: 1,             // Epoch number
  cacheSize: 16907968,  // Cache size in bytes
  datasetSize: 1082128512 // Dataset size in bytes
}
```

### `verifyProgPoW(headerHash, nonce, blockNumber, primeTerminusNumber, expectedMixHash, difficulty)`
Verifies a ProgPoW solution against expected mix hash and difficulty.

**Returns:**
```javascript
{
  valid: true,          // Overall validity
  mixHashValid: true,   // Mix hash matches
  powValid: true,       // Meets difficulty
  computedMixHash: "0x...",
  powHash: "0x...",
  target: "0x...",
  difficulty: "1000000"
}
```

### `computeWorkObjectSealHash(headerData)`
Computes the seal hash for a work object header using protobuf encoding and Blake3.

**Parameters:**
- `headerData` - Object with header fields (headerHash, parentHash, number, etc.)

**Returns:**
```javascript
{
  sealHash: "0x...",    // 32-byte hex
  protoSize: 167,       // Protobuf size in bytes
  protoHex: "0x..."     // Encoded protobuf hex
}
```

### `verifyWithExactSealHash(headerData)`
Performs full verification including seal hash computation.

**Returns:**
```javascript
{
  valid: true,
  sealHash: "0x...",
  mixHashValid: true,
  powValid: true,
  computedMixHash: "0x...",
  expectedMixHash: "0x...",
  powHash: "0x...",
  target: "0x...",
  difficulty: "1000000"
}
```

## ⚙️ Algorithm Details

### ProgPoW Parameters
- **Algorithm**: ProgPoW (Programmatic Proof-of-Work)
- **Epoch Length**: 388,800 blocks (C_epochLength)
- **Cache Rounds**: 4 (PROGPOW_CNT_DAG)
- **Hash Function**: Keccak-256 for ProgPoW, Blake3 for seal hash

### Cache Generation
- Epoch-based cache management
- Light verification using cache (no full dataset required)
- Cache size formula: `cacheSize(epoch*C_epochLength + 1)`
- Dataset size formula: `datasetSize(epoch*C_epochLength + 1)`

### Seal Hash
The seal hash is computed by:
1. Creating a protobuf-encoded WorkObjectHeader (excluding mixHash and nonce)
2. Hashing the encoded data with Blake3
3. This matches Quai's `WorkObjectHeader.SealEncode()` implementation

## 🔧 Module Dependencies

- `golang.org/x/crypto` - SHA3/Keccak hashing
- `google.golang.org/protobuf` - Protocol buffer support
- `lukechampine.com/blake3` - Blake3 hashing for seal hash
- `github.com/sirupsen/logrus` - Structured logging

## 📈 Performance

- **Cache Generation**: ~100ms for epoch 1
- **Verification**: ~50-200ms per block (browser)
- **Memory Usage**: ~20MB for epoch 1 cache
- **WASM Size**: ~2.4MB compiled

## 🔒 Security Notes

- This implementation is for **verification only**, not mining
- All cryptographic operations use standard libraries
- No private keys or sensitive data are handled
- Suitable for light clients and browser verification

## 📄 License

LGPL-3.0 (inherited from go-ethereum)