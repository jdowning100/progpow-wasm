#!/usr/bin/env node

const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = process.env.PORT || 8090;

// Set proper MIME types and handle compressed files
app.use((req, res, next) => {
    // Handle compressed WASM files
    if (req.path.endsWith('.wasm.gz')) {
        res.set({
            'Content-Type': 'application/wasm',
            'Content-Encoding': 'gzip'
        });
    } else if (req.path.endsWith('.wasm')) {
        // Check if compressed version exists and client accepts gzip
        const gzPath = path.join(__dirname, req.path + '.gz');
        if (fs.existsSync(gzPath) && req.get('Accept-Encoding')?.includes('gzip')) {
            res.set({
                'Content-Type': 'application/wasm',
                'Content-Encoding': 'gzip'
            });
            return res.sendFile(gzPath);
        }
        res.type('application/wasm');
    }
    next();
});

// Handle both direct access and proxy access through /progpow
app.use('/progpow', express.static(__dirname));
app.use('/', express.static(__dirname));

// Redirect /progpow to /progpow/ for proper relative paths
app.get('/progpow', (req, res) => {
    res.redirect('/progpow/');
});

// SPA root redirects
app.get('/', (req, res) => {
    res.redirect('/test_progpow.html');
});

app.get('/progpow/', (req, res) => {
    res.sendFile(path.join(__dirname, 'test_progpow.html'));
});

// Start server
app.listen(PORT, 'localhost', () => {
    console.log('🚀 ProgPoW WASM Server running');
    console.log(`📍 Local: http://localhost:${PORT}`);
    console.log(`📄 Direct access: http://localhost:${PORT}/test_progpow.html`);
    console.log(`📄 Proxy access: http://localhost:${PORT}/progpow/`);
    console.log('\nPress Ctrl+C to stop the server');
});