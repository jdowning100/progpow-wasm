//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"syscall/js"

	"lukechampine.com/blake3"
	"google.golang.org/protobuf/proto"
)

// computeWorkObjectSealHash computes the exact seal hash for a WorkObjectHeader
// This matches the implementation in wo.go
func computeWorkObjectSealHash(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "Missing header object",
		}
	}

	headerObj := args[0]
	
	// Helper to extract hash from JS object
	extractHash := func(fieldName string) *ProtoHash {
		if headerObj.Get(fieldName).Type() != js.TypeUndefined {
			value := headerObj.Get(fieldName).String()
			// Remove 0x prefix if present
			if len(value) > 2 && value[:2] == "0x" {
				value = value[2:]
			}
			// Pad with leading zero if odd length
			if len(value)%2 == 1 {
				value = "0" + value
			}
			if bytes, err := hex.DecodeString(value); err == nil {
				return &ProtoHash{Value: bytes}
			}
		}
		return &ProtoHash{Value: []byte{}}
	}
	
	// Helper to extract bytes from hex string
	extractBytes := func(fieldName string) []byte {
		if headerObj.Get(fieldName).Type() != js.TypeUndefined {
			value := headerObj.Get(fieldName).String()
			// Remove 0x prefix if present
			if len(value) > 2 && value[:2] == "0x" {
				value = value[2:]
			}
			// Pad with leading zero if odd length
			if len(value)%2 == 1 {
				value = "0" + value
			}
			if bytes, err := hex.DecodeString(value); err == nil {
				return bytes
			} else {
			}
		} else {
		}
		return nil  // Return nil instead of empty slice for optional fields
	}
	
	// Helper to extract address
	extractAddress := func(fieldName string) *ProtoAddress {
		if headerObj.Get(fieldName).Type() != js.TypeUndefined {
			value := headerObj.Get(fieldName).String()
			// Remove 0x prefix if present
			if len(value) > 2 && value[:2] == "0x" {
				value = value[2:]
			}
			// Pad with leading zero if odd length
			if len(value)%2 == 1 {
				value = "0" + value
			}
			if bytes, err := hex.DecodeString(value); err == nil {
				return &ProtoAddress{Value: bytes}
			}
		}
		return &ProtoAddress{Value: []byte{}}
	}
	
	// Helper to extract uint32
	extractUint32 := func(fieldName string) uint32 {
		if headerObj.Get(fieldName).Type() != js.TypeUndefined {
			return uint32(headerObj.Get(fieldName).Int())
		}
		return 0
	}
	
	// Helper to extract uint64
	extractUint64 := func(fieldName string) uint64 {
		if headerObj.Get(fieldName).Type() != js.TypeUndefined {
			value := headerObj.Get(fieldName).String()
			if len(value) > 2 && value[:2] == "0x" {
				// Parse hex string
				big := new(big.Int)
				big.SetString(value[2:], 16)
				return big.Uint64()
			}
			// Try as number
			return uint64(headerObj.Get(fieldName).Float())
		}
		return 0
	}
	
	// Extract location
	var location *ProtoLocation
	if headerObj.Get("location").Type() != js.TypeUndefined {
		locBytes := extractBytes("location")
		location = &ProtoLocation{Value: locBytes}
	}
	
	// Build ProtoWorkObjectHeader for SealEncode (excluding MixHash and Nonce)
	lock := extractUint32("lock")
	time := extractUint64("time")
	
	numberBytes := extractBytes("number")
	
	protoHeader := &ProtoWorkObjectHeader{
		HeaderHash:          extractHash("headerHash"),
		ParentHash:          extractHash("parentHash"),
		Number:              numberBytes,
		Difficulty:          extractBytes("difficulty"),
		TxHash:              extractHash("txHash"),
		PrimeTerminusNumber: extractBytes("primeTerminusNumber"),
		Location:            location,
		Lock:                &lock,
		PrimaryCoinbase:     extractAddress("primaryCoinbase"),
		Time:                &time,
		Data:                extractBytes("data"),
		// Note: MixHash and Nonce are explicitly excluded for seal hash
	}
	
	// Marshal to protobuf
	data, err := proto.Marshal(protoHeader)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to marshal protobuf: %v", err),
		}
	}
	
	// Compute Blake3 hash
	sum := blake3.Sum256(data)
	
	return map[string]interface{}{
		"sealHash": hex.EncodeToString(sum[:]),
		"protoSize": len(data),
		"protoHex": hex.EncodeToString(data),
	}
}

// verifyWithExactSealHash verifies ProgPoW using the exact seal hash computation
func verifyWithExactSealHash(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "Missing header object",
		}
	}
	
	// First compute the exact seal hash
	sealHashResult := computeWorkObjectSealHash(this, args)
	sealHashMap := sealHashResult.(map[string]interface{})
	
	if sealHashMap["error"] != nil {
		return sealHashResult
	}
	
	sealHash := sealHashMap["sealHash"].(string)
	headerObj := args[0]
	
	// Extract nonce
	nonce := uint64(0)
	if headerObj.Get("nonce").Type() != js.TypeUndefined {
		nonceVal := headerObj.Get("nonce")
		if nonceVal.Type() == js.TypeString {
			nonceStr := nonceVal.String()
			if len(nonceStr) > 2 && nonceStr[:2] == "0x" {
				big := new(big.Int)
				big.SetString(nonceStr[2:], 16)
				nonce = big.Uint64()
			} else {
				// Try to parse as decimal string
				big := new(big.Int)
				big.SetString(nonceStr, 10)
				nonce = big.Uint64()
			}
		} else {
			nonce = uint64(nonceVal.Float())
		}
	}
	
	// Extract block number
	blockNumber := uint64(0)
	if headerObj.Get("number").Type() != js.TypeUndefined {
		numVal := headerObj.Get("number")
		if numVal.Type() == js.TypeString {
			numStr := numVal.String()
			if len(numStr) > 2 && numStr[:2] == "0x" {
				big := new(big.Int)
				big.SetString(numStr[2:], 16)
				blockNumber = big.Uint64()
			} else {
				// Try to parse as decimal string
				big := new(big.Int)
				big.SetString(numStr, 10)
				blockNumber = big.Uint64()
			}
		} else {
			blockNumber = uint64(numVal.Float())
		}
	}
	
	// Extract prime terminus
	primeTerminus := uint64(0)
	if headerObj.Get("primeTerminusNumber").Type() != js.TypeUndefined {
		ptVal := headerObj.Get("primeTerminusNumber")
		if ptVal.Type() == js.TypeString {
			ptStr := ptVal.String()
			if len(ptStr) > 2 && ptStr[:2] == "0x" {
				big := new(big.Int)
				big.SetString(ptStr[2:], 16)
				primeTerminus = big.Uint64()
			} else {
				// Try to parse as decimal string
				big := new(big.Int)
				big.SetString(ptStr, 10)
				primeTerminus = big.Uint64()
			}
		} else {
			primeTerminus = uint64(ptVal.Float())
		}
	}
	
	// Extract expected mix hash
	expectedMixHash := ""
	if headerObj.Get("mixHash").Type() != js.TypeUndefined {
		expectedMixHash = headerObj.Get("mixHash").String()
		if len(expectedMixHash) > 2 && expectedMixHash[:2] == "0x" {
			expectedMixHash = expectedMixHash[2:]
		}
	}
	
	// Extract difficulty
	difficulty := new(big.Int).SetInt64(1)
	if headerObj.Get("difficulty").Type() != js.TypeUndefined {
		diffVal := headerObj.Get("difficulty")
		if diffVal.Type() == js.TypeString {
			diffStr := diffVal.String()
			if len(diffStr) > 2 && diffStr[:2] == "0x" {
				difficulty.SetString(diffStr[2:], 16)
			} else {
				difficulty.SetString(diffStr, 10)
			}
		} else {
			difficulty.SetInt64(int64(diffVal.Float()))
		}
	}
	
	// Call verifyProgPoW with the computed seal hash
	// Pass nonce as hex string to preserve precision
	nonceHex := fmt.Sprintf("0x%016x", nonce)
	newArgs := []js.Value{
		js.ValueOf(sealHash),
		js.ValueOf(nonceHex),  // Pass as hex string to avoid JS number precision loss
		js.ValueOf(blockNumber),
		js.ValueOf(primeTerminus),
		js.ValueOf(expectedMixHash),
		js.ValueOf(difficulty.Uint64()),
	}
	
	result := verifyProgPoW(this, newArgs)
	resultMap := result.(map[string]interface{})
	
	// Add the seal hash to the result
	resultMap["sealHash"] = sealHash
	
	return resultMap
}

