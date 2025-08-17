#!/bin/bash

# Simple HTTP server to test ProgPoW WASM module
# Serves files with proper MIME types for WASM

echo "Starting HTTP server for ProgPoW WASM testing..."
echo ""
echo "Test pages available at:"
echo "  - http://localhost:8080/test_progpow.html"
echo ""
echo "Press Ctrl+C to stop the server"

# Use Python's built-in HTTP server with proper MIME types
python3 -m http.server 8080 --bind localhost