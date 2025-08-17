#!/bin/bash

# Build script for ProgPoW WASM module
# This script builds the WebAssembly module that exports ProgPoW verification functions to JavaScript

set -e

echo "🔨 Building ProgPoW WASM module..."

# Ensure we're in the correct directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Build the WASM module
echo "📦 Compiling Go to WebAssembly..."
cd cmd
GOOS=js GOARCH=wasm go build -o ../progpow_full.wasm .
cd ..

# Check if build succeeded
if [ -f "progpow_full.wasm" ]; then
    SIZE=$(ls -lh progpow_full.wasm | awk '{print $5}')
    echo "✅ Build successful! Output: progpow_full.wasm (${SIZE})"
    echo ""
    echo "📚 Exported functions:"
    echo "  - computeProgPoW(headerHash, nonce, blockNumber, primeTerminusNumber)"
    echo "  - verifyProgPoW(headerHash, nonce, blockNumber, primeTerminusNumber, expectedMixHash, difficulty)"
    echo "  - computeWorkObjectSealHash(headerData)"
    echo "  - verifyWithExactSealHash(headerData)"
    echo ""
    echo "🧪 To test:"
    echo "  - Run: ./serve.sh"
    echo "  - Open: http://localhost:8080/test_progpow_updated.html"
    echo "  - Or run: node verify_block.js [block_number]"
else
    echo "❌ Build failed!"
    exit 1
fi