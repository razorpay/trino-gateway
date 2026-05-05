#!/usr/bin/env node

/**
 * HTTP Client Helper for Payment Integration Tests
 * Usage: node http-client.js <method> <url> [body] [username:password]
 *
 * Examples:
 *   node http-client.js GET http://localhost:3001/api/orders/ord_123 "" "test:test"
 *   node http-client.js POST http://localhost:3001/api/orders '{"amount":50000}' "test:test"
 */

const https = require('https');
const http = require('http');

const [method, url, bodyStr, authStr] = process.argv.slice(2);

if (!method || !url) {
  console.error('Usage: node http-client.js <method> <url> [body] [username:password]');
  process.exit(1);
}

const urlObj = new URL(url);
const client = urlObj.protocol === 'https:' ? https : http;

const options = {
  method: method.toUpperCase(),
  hostname: urlObj.hostname,
  port: urlObj.port || (urlObj.protocol === 'https:' ? 443 : 80),
  path: urlObj.pathname + urlObj.search,
  headers: {
    'Content-Type': 'application/json'
  }
};

// Add Basic Auth if provided
if (authStr) {
  const auth = Buffer.from(authStr).toString('base64');
  options.headers['Authorization'] = `Basic ${auth}`;
}

const req = client.request(options, (res) => {
  let data = '';

  res.on('data', (chunk) => {
    data += chunk;
  });

  res.on('end', () => {
    try {
      const response = {
        status: res.statusCode,
        headers: res.headers,
        body: data ? JSON.parse(data) : null
      };
      console.log(JSON.stringify(response, null, 2));
      process.exit(res.statusCode >= 400 ? 1 : 0);
    } catch (e) {
      console.log(JSON.stringify({
        status: res.statusCode,
        headers: res.headers,
        body: data
      }, null, 2));
      process.exit(res.statusCode >= 400 ? 1 : 0);
    }
  });
});

req.on('error', (error) => {
  console.error(JSON.stringify({
    error: error.message,
    code: error.code
  }, null, 2));
  process.exit(1);
});

// Write request body if provided
if (bodyStr && bodyStr !== '""' && bodyStr !== "''") {
  req.write(bodyStr);
}

req.end();
