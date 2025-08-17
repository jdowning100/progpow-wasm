//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"syscall/js"

	"github.com/dominant-strategies/progpow-wasm/progpow"
	"github.com/sirupsen/logrus"
)

// C_epochLength is the epoch length for ProgPoW
const C_epochLength = 388800

// Create a logger for WASM
var logger = logrus.New()

func main() {
	fmt.Println("ProgPoW WASM module initializing...")

	// Configure logger for WASM environment
	logger.SetLevel(logrus.InfoLevel) // Only show warnings and errors
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: false,
	})

	// Register functions that JavaScript can call
	js.Global().Set("progpowInfo", js.FuncOf(progpowInfo))
	js.Global().Set("verifyProgPoW", js.FuncOf(verifyProgPoW))
	js.Global().Set("computeProgPoW", js.FuncOf(computeProgPoW))
	js.Global().Set("computeWorkObjectSealHash", js.FuncOf(computeWorkObjectSealHash))
	js.Global().Set("verifyWithExactSealHash", js.FuncOf(verifyWithExactSealHash))

	// Signal that the module is ready
	js.Global().Set("progpowReady", true)

	fmt.Println("ProgPoW WASM module ready")

	// Keep the program running
	select {}
}

// progpowInfo returns basic information about the ProgPoW algorithm
func progpowInfo(this js.Value, args []js.Value) interface{} {
	return map[string]interface{}{
		"algorithm":   "ProgPoW",
		"version":     "0.9.4",
		"cacheSize":   16384, // 16KB
		"epochLength": C_epochLength,
		"dagLoads":    4,
		"lanes":       16,
		"regs":        32,
		"description": "GPU-friendly, ASIC-resistant proof of work",
	}
}

// computeProgPoW computes the ProgPoW hash for a given header
// Args: headerHash (hex), nonce (number), blockNumber (number), primeTerminusNumber (number)
// Returns: {mixHash: hex, powHash: hex}
func computeProgPoW(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		return map[string]interface{}{
			"error": "Expected: headerHash (hex), nonce (number), blockNumber (number), primeTerminusNumber (number)",
		}
	}

	// Parse arguments
	headerHashHex := args[0].String()
	
	// Parse nonce - can be a number or string to handle large values
	var nonce uint64
	if args[1].Type() == js.TypeString {
		nonceStr := args[1].String()
		if len(nonceStr) > 2 && nonceStr[:2] == "0x" {
			nonceStr = nonceStr[2:]
		}
		nonceBig := new(big.Int)
		nonceBig.SetString(nonceStr, 16)
		nonce = nonceBig.Uint64()
	} else {
		// For smaller nonces that fit in JavaScript number
		nonce = uint64(args[1].Float())
	}
	
	_ = uint64(args[2].Float()) // blockNumber - not used, we use primeTerminusNumber for progpowLight  
	primeTerminusNumber := uint64(args[3].Float())

	// Remove 0x prefix if present
	if len(headerHashHex) > 2 && headerHashHex[:2] == "0x" {
		headerHashHex = headerHashHex[2:]
	}

	// Decode header hash
	headerHash, err := hex.DecodeString(headerHashHex)
	if err != nil || len(headerHash) != 32 {
		return map[string]interface{}{
			"error": "Invalid header hash - must be 32 bytes hex",
		}
	}

	// Calculate epoch and cache size
	// Note: primeTerminusNumber is treated like a block number for epoch calculation
	epoch := primeTerminusNumber / C_epochLength
	// CacheSize and DatasetSize expect a block number, not epoch
	cacheSize := progpow.CacheSize(epoch*C_epochLength + 1)
	datasetSize := progpow.DatasetSize(epoch*C_epochLength + 1)

	// Generate seed for the epoch
	// Note: seedHash expects a block number, not epoch number
	seed := progpow.SeedHash(epoch*C_epochLength + 1)

	// Generate cache
	cache := make([]uint32, cacheSize/4)
	progpow.GenerateCache(cache, epoch, seed, logger)

	// Generate cDag
	cDag := make([]uint32, 16*1024/4) // progpowCacheBytes / 4
	progpow.GenerateCDag(cDag, cache, epoch, logger)

	
	// Compute ProgPoW
	// Note: Go implementation passes primeTerminusNumber as the blockNumber to progpowLight
	mixHash, powHash := progpow.ProgpowLight(datasetSize, cache, headerHash, nonce, primeTerminusNumber, cDag)

	return map[string]interface{}{
		"mixHash":     hex.EncodeToString(mixHash),
		"powHash":     hex.EncodeToString(powHash),
		"epoch":       epoch,
		"cacheSize":   cacheSize,
		"datasetSize": datasetSize,
	}
}

// verifyProgPoW performs FULL verification of a ProgPoW hash
// This recomputes the mixHash and powHash from scratch and verifies both:
// 1. The provided mixHash matches the computed one
// 2. The powHash meets the difficulty target
// Args: headerHash (hex), nonce (number), blockNumber (number), primeTerminusNumber (number), mixHash (hex), difficulty (number or hex)
// Returns: {valid: bool, powHash: hex, target: hex}
func verifyProgPoW(this js.Value, args []js.Value) interface{} {
	if len(args) < 6 {
		return map[string]interface{}{
			"error": "Expected: headerHash (hex), nonce (number), blockNumber (number), primeTerminusNumber (number), mixHash (hex), difficulty (number or hex)",
		}
	}

	// Parse arguments
	headerHashHex := args[0].String()
	
	// Parse nonce - can be a number or string to handle large values
	var nonce uint64
	if args[1].Type() == js.TypeString {
		nonceStr := args[1].String()
		if len(nonceStr) > 2 && nonceStr[:2] == "0x" {
			nonceStr = nonceStr[2:]
		}
		nonceBig := new(big.Int)
		nonceBig.SetString(nonceStr, 16)
		nonce = nonceBig.Uint64()
	} else {
		// For smaller nonces that fit in JavaScript number
		nonce = uint64(args[1].Float())
	}
	
	blockNumber := uint64(args[2].Float())
	primeTerminusNumber := uint64(args[3].Float())
	expectedMixHashHex := args[4].String()

	// Parse difficulty - can be number or hex string
	var difficulty *big.Int
	if args[5].Type() == js.TypeNumber {
		difficulty = big.NewInt(int64(args[5].Float()))
	} else {
		difficultyStr := args[5].String()
		if len(difficultyStr) > 2 && difficultyStr[:2] == "0x" {
			difficultyStr = difficultyStr[2:]
		}
		difficulty = new(big.Int)
		difficulty.SetString(difficultyStr, 16)
	}

	// Remove 0x prefix if present
	if len(headerHashHex) > 2 && headerHashHex[:2] == "0x" {
		headerHashHex = headerHashHex[2:]
	}
	if len(expectedMixHashHex) > 2 && expectedMixHashHex[:2] == "0x" {
		expectedMixHashHex = expectedMixHashHex[2:]
	}

	// Decode hex strings
	headerHash, err := hex.DecodeString(headerHashHex)
	if err != nil || len(headerHash) != 32 {
		return map[string]interface{}{
			"error": "Invalid header hash - must be 32 bytes hex",
		}
	}

	expectedMixHash, err := hex.DecodeString(expectedMixHashHex)
	if err != nil || len(expectedMixHash) != 32 {
		return map[string]interface{}{
			"error": "Invalid mix hash - must be 32 bytes hex",
		}
	}

	// Compute ProgPoW
	// Pass nonce as hex string to preserve precision
	nonceHex := fmt.Sprintf("0x%016x", nonce)
	result := computeProgPoW(this, []js.Value{
		js.ValueOf(headerHashHex),
		js.ValueOf(nonceHex),  // Pass as hex string to avoid precision loss
		js.ValueOf(blockNumber),
		js.ValueOf(primeTerminusNumber),
	})

	// Check if computation had error
	resultMap := result.(map[string]interface{})
	if err, hasError := resultMap["error"]; hasError {
		return map[string]interface{}{
			"error": err,
		}
	}

	// Get computed hashes
	computedMixHash := resultMap["mixHash"].(string)
	computedPowHash := resultMap["powHash"].(string)

	// Verify mix hash matches
	mixHashValid := computedMixHash == expectedMixHashHex

	// Calculate target from difficulty
	// target = 2^256 / difficulty
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	target := new(big.Int).Div(two256, difficulty)

	// Convert powHash to big.Int for comparison
	powHashBytes, _ := hex.DecodeString(computedPowHash)
	powHashInt := new(big.Int).SetBytes(powHashBytes)

	// Check if powHash <= target (valid if true)
	powValid := powHashInt.Cmp(target) <= 0

	return map[string]interface{}{
		"valid":           mixHashValid && powValid,
		"mixHashValid":    mixHashValid,
		"powValid":        powValid,
		"computedMixHash": computedMixHash,
		"expectedMixHash": expectedMixHashHex,
		"powHash":         computedPowHash,
		"target":          hex.EncodeToString(target.Bytes()),
		"difficulty":      difficulty.String(),
	}
}
