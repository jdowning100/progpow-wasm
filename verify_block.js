#!/usr/bin/env node

/**
 * ProgPoW Block Verification using WASM
 * 
 * This script fetches blocks from a Quai network node and verifies them
 * using the ProgPoW WASM implementation.
 * 
 * Usage:
 *   node verify_block.js                    # Verify latest block
 *   node verify_block.js 1000               # Verify specific block
 *   node verify_block.js --range 1 10       # Verify block range
 */

const fs = require('fs');
const fetch = require('node-fetch');

// Configuration
const RPC_ENDPOINT = process.env.QUAI_RPC || 'https://rpc.quai.network/cyprus1';

// Colors for terminal output
const colors = {
    reset: '\x1b[0m',
    bright: '\x1b[1m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    cyan: '\x1b[36m'
};

// Load WASM module
async function loadWASM() {
    console.log(`${colors.cyan}Loading ProgPoW WASM module...${colors.reset}`);
    
    // Load the wasm_exec.js runtime
    const wasmExecCode = fs.readFileSync('./wasm_exec.js', 'utf8');
    eval(wasmExecCode);
    
    // Create Go instance
    const go = new Go();
    
    // Load WASM file
    const wasmBuffer = fs.readFileSync('./progpow_full.wasm');
    const wasmModule = await WebAssembly.instantiate(wasmBuffer, go.importObject);
    
    // Run the Go program
    go.run(wasmModule.instance);
    
    // Wait for initialization
    await new Promise(resolve => setTimeout(resolve, 100));
    
    if (global.progpowReady) {
        console.log(`${colors.green}✓ WASM module loaded successfully${colors.reset}`);
        return true;
    } else {
        throw new Error('WASM module failed to initialize');
    }
}

// Fetch block from RPC
async function fetchBlock(blockNumber = null) {
    const params = blockNumber !== null ? 
        [`0x${blockNumber.toString(16)}`, false] : 
        ['latest', false];

    const response = await fetch(RPC_ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            jsonrpc: '2.0',
            method: 'quai_getBlockByNumber',
            params: params,
            id: 1
        })
    });

    if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = await response.json();
    if (data.error) {
        throw new Error(`RPC Error: ${data.error.message}`);
    }

    return data.result;
}

// Compute seal hash (simplified version for demo)
function computeSealHash(block) {
    // In production, this would compute the actual Keccak256 seal hash
    // For this demo, we'll use the headerHash from woHeader if available
    const woHeader = block.woHeader || block.workObjectHeader || block;
    
    // If we have headerHash, use it as the seal hash
    if (woHeader.headerHash) {
        return woHeader.headerHash.replace('0x', '');
    }
    
    // Otherwise create a deterministic hash
    const parentHashClean = (woHeader.parentHash || '0x0').replace('0x', '');
    const blockNumHex = (woHeader.number || '0x0').replace('0x', '').padStart(16, '0');
    
    // Combine parent hash with block number for a pseudo seal hash
    return parentHashClean.substring(0, 48) + blockNumHex;
}

// Verify a block
async function verifyBlock(block) {
    // Extract work object header from block structure
    const woHeader = block.woHeader || block.workObjectHeader || block;
    
    // Use work object header fields for ProgPoW
    const blockNumber = parseInt(woHeader.number, 16);
    const difficulty = parseInt(woHeader.difficulty, 16);
    const primeTerminus = parseInt(woHeader.primeTerminusNumber || '0x0', 16);
    const nonceHex = (woHeader.nonce || '0x0').replace('0x', '');
    const nonce = parseInt(nonceHex, 16);
    const mixHash = (woHeader.mixHash || '0x0').replace('0x', '');
    
    console.log(`\n${colors.bright}Block #${blockNumber}${colors.reset}`);
    console.log(`  Hash: ${block.hash}`);
    console.log(`  Difficulty: ${difficulty}`);
    console.log(`  Nonce: 0x${nonceHex}`);
    console.log(`  Mix Hash: ${mixHash}`);
    console.log(`  Prime Terminus: ${primeTerminus}`);
    
    // Compute seal hash
    const sealHash = computeSealHash(block);
    console.log(`  Seal Hash: ${sealHash}`);
    
    // Perform ProgPoW verification
    console.log(`\n${colors.cyan}Running ProgPoW verification...${colors.reset}`);
    
    const result = global.verifyProgPoW(
        sealHash,
        nonce,
        blockNumber,
        primeTerminus,
        mixHash,
        difficulty  // Use the parsed difficulty number, not the hex string
    );
    
    if (result.error) {
        throw new Error(`Verification error: ${result.error}`);
    }
    
    // Display results
    console.log('\nVerification Results:');
    console.log(`  Mix Hash Valid: ${result.mixHashValid ? colors.green + '✓' : colors.red + '✗'}${colors.reset}`);
    console.log(`  PoW Valid: ${result.powValid ? colors.green + '✓' : colors.red + '✗'}${colors.reset}`);
    console.log(`  Overall: ${result.valid ? colors.green + 'VALID' : colors.red + 'INVALID'}${colors.reset}`);
    
    if (!result.mixHashValid) {
        console.log(`\n${colors.yellow}Mix Hash Mismatch:${colors.reset}`);
        console.log(`  Computed: ${result.computedMixHash}`);
        console.log(`  Expected: ${result.expectedMixHash}`);
    }
    
    console.log(`\n${colors.cyan}PoW Details:${colors.reset}`);
    console.log(`  PoW Hash: ${result.powHash}`);
    console.log(`  Target:   ${result.target}`);
    console.log(`  Difficulty: ${result.difficulty}`);
    
    return result.valid;
}

// Verify a range of blocks
async function verifyBlockRange(start, end) {
    let validCount = 0;
    let invalidCount = 0;
    const results = [];
    
    console.log(`\n${colors.bright}Verifying blocks ${start} to ${end}${colors.reset}`);
    
    for (let blockNum = start; blockNum <= end; blockNum++) {
        try {
            console.log(`\n${'='.repeat(60)}`);
            const block = await fetchBlock(blockNum);
            const isValid = await verifyBlock(block);
            
            if (isValid) {
                validCount++;
                results.push({ block: blockNum, status: 'VALID' });
            } else {
                invalidCount++;
                results.push({ block: blockNum, status: 'INVALID' });
            }
        } catch (error) {
            console.error(`${colors.red}Error verifying block ${blockNum}: ${error.message}${colors.reset}`);
            results.push({ block: blockNum, status: 'ERROR', error: error.message });
        }
    }
    
    // Summary
    console.log(`\n${'='.repeat(60)}`);
    console.log(`${colors.bright}Verification Summary:${colors.reset}`);
    console.log(`  Total blocks: ${end - start + 1}`);
    console.log(`  Valid: ${colors.green}${validCount}${colors.reset}`);
    console.log(`  Invalid: ${colors.red}${invalidCount}${colors.reset}`);
    console.log(`  Errors: ${colors.yellow}${results.filter(r => r.status === 'ERROR').length}${colors.reset}`);
    
    // Detailed results
    console.log(`\n${colors.cyan}Detailed Results:${colors.reset}`);
    results.forEach(r => {
        const statusColor = r.status === 'VALID' ? colors.green : 
                          r.status === 'INVALID' ? colors.red : colors.yellow;
        console.log(`  Block #${r.block}: ${statusColor}${r.status}${colors.reset}${r.error ? ' - ' + r.error : ''}`);
    });
}

// Main function
async function main() {
    try {
        console.log(`${colors.bright}ProgPoW Block Verifier${colors.reset}`);
        console.log(`RPC Endpoint: ${RPC_ENDPOINT}`);
        
        // Load WASM module
        await loadWASM();
        
        // Parse command line arguments
        const args = process.argv.slice(2);
        
        if (args.length === 0) {
            // Verify latest block
            console.log(`\n${colors.cyan}Fetching latest block...${colors.reset}`);
            const block = await fetchBlock();
            await verifyBlock(block);
            
        } else if (args[0] === '--range' && args.length === 3) {
            // Verify block range
            const start = parseInt(args[1]);
            const end = parseInt(args[2]);
            
            if (isNaN(start) || isNaN(end) || start < 0 || end < start) {
                throw new Error('Invalid block range');
            }
            
            await verifyBlockRange(start, end);
            
        } else if (args.length === 1 && !isNaN(parseInt(args[0]))) {
            // Verify specific block
            const blockNumber = parseInt(args[0]);
            console.log(`\n${colors.cyan}Fetching block #${blockNumber}...${colors.reset}`);
            const block = await fetchBlock(blockNumber);
            await verifyBlock(block);
            
        } else {
            console.log('\nUsage:');
            console.log('  node verify_block.js                    # Verify latest block');
            console.log('  node verify_block.js 1000               # Verify specific block');
            console.log('  node verify_block.js --range 1 10       # Verify block range');
            process.exit(1);
        }
        
        console.log(`\n${colors.green}✓ Verification complete${colors.reset}`);
        
    } catch (error) {
        console.error(`\n${colors.red}Error: ${error.message}${colors.reset}`);
        if (error.stack) {
            console.error(colors.red + error.stack + colors.reset);
        }
        process.exit(1);
    }
}

// Check if node-fetch is installed
try {
    require.resolve('node-fetch');
} catch (e) {
    console.error(`${colors.red}Error: node-fetch is not installed${colors.reset}`);
    console.log('Please install it with: npm install node-fetch');
    process.exit(1);
}

// Run the main function
main();