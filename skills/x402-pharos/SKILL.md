---
name: x402-pharos
description: x402 payment protocol for Pharos testnet. Use when building paid APIs, monetizing endpoints, or integrating crypto payments on Pharos. Triggers on "x402 pharos", "paid API pharos", "pharos payment", "payment gateway pharos".
license: MIT
metadata:
  author: antdigital
  version: "1.0.0"
---

# x402 Pharos Payment Protocol

x402 is an open standard for HTTP-native payments, adapted for Pharos testnet.

## Pharos Network

- **Chain ID**: 688689
- **RPC URL**: https://atlantic.dplabs-internal.com
- **USDC Address**: 0xYourTokenAddress (provided by user)
- **Network Identifier**: `eip155:688689`
- **Facilitator URL**: need provided by user

## When to Use

- Building paid APIs on Pharos testnet
- Adding payment requirements to existing services
- Creating AI agents that can pay for API access using USDC
- Integrating crypto payments into web applications on Pharos

---

## Quick Start

### Server Setup

```bash
# Create workspace
mkdir my-x402-server && cd my-x402-server
npm init -y
npm install @x402/core @x402/express @x402/evm @x402/fetch express viem typescript tsx @types/node @types/express dotenv
npx tsc --init --esModuleInterop --moduleResolution node --module esnext --target es2022
```

Create `.env` file:
```bash
# Required: Your receiving address
PAY_TO_ADDRESS=0x your receiving address

# Optional: Port configuration
PORT=4021

# Network configuration
FACILITATOR_URL=http://xxx your facilitator url

# USDC address (Pharos testnet)
USDC_ADDRESS=0xYourTokenAddress

# USDC name (optional)
USDC_NAME=USDC
```

Run the server:
```bash
npx tsx server.ts
```

> **⚠️ Important**: Make sure to run commands from the project directory. If you encounter `ERR_MODULE_NOT_FOUND` errors, verify:
> 1. `node_modules` is installed in the current directory (not parent directory)
> 2. You are running `npx tsx server.ts` from the same directory where you ran `npm install`
> 3. Run `npm install` again in the current directory if needed

### Client Setup

```bash
# Create workspace
mkdir my-x402-client && cd my-x402-client
npm init -y
npm install @x402/core @x402/fetch @x402/evm viem dotenv tsx typescript @types/node
```

Create `.env` file:
```bash
# Required: Your private key (for payments)
EVM_PRIVATE_KEY=0x your private key here

# Optional: Server address
SERVER_URL=http://localhost:4021
```

Run the client:
```bash
npx tsx client.ts http://localhost:4021/data
```

---

## Configuration

### Environment Variables

**Client:**
- `EVM_PRIVATE_KEY`: Private key (set via environment variable or file)
- `SERVER_URL`: Server url

**Server:**
- `PAY_TO_ADDRESS`: Receiving address
- `PORT`: Server port (optional)
- `FACILITATOR_URL`: Facilitator address
- `USDC_ADDRESS`: USDC token address
- `USDC_NAME`: USDC token name (optional, defaults to "USDC")

### Security Best Practices

**❌ Don't do:**
- Don't write private keys directly in code
- Don't commit private keys to version control
- Don't log private keys

**✅ Recommended practices:**
- Use environment variables for private keys
- Use `.private_key` file (added to .gitignore)
- Use `.env` file (added to .gitignore)

---

## Complete Code Examples

### Server (Seller) - Monetize Your API

```typescript
// server.ts - Complete working implementation
import express from "express";
import { paymentMiddleware, x402ResourceServer } from "@x402/express";
import { ExactEvmScheme } from "@x402/evm/exact/server";
import { HTTPFacilitatorClient } from "@x402/core/server";
import { config } from "dotenv";

// Load environment variables
config();

// Securely get configuration - read from environment variables
const payToAddress = process.env.PAY_TO_ADDRESS as `0x${string}`;
if (!payToAddress) {
  console.error('Please set PAY_TO_ADDRESS environment variable');
  process.exit(1);
}

if (!payToAddress.startsWith('0x') || payToAddress.length !== 42) {
  console.error('Invalid receiving address format');
  process.exit(1);
}

const facilitatorUrl = process.env.FACILITATOR_URL;
const port = process.env.PORT || 4021;
const usdcAddress = process.env.USDC_ADDRESS;
const usdcName = process.env.USDC_NAME || "USDC";

if (!facilitatorUrl || !usdcAddress) {
  console.error('Please set FACILITATOR_URL and USDC_ADDRESS');
  process.exit(1);
}

// Create Facilitator client
const facilitatorClient = new HTTPFacilitatorClient({ url: facilitatorUrl });

// Initialize x402 resource server
const resourceServer = new x402ResourceServer(facilitatorClient);

// Create EVM scheme instance
const evmScheme = new ExactEvmScheme();

// Register custom USDC configuration for Pharos network
evmScheme.registerMoneyParser(async (amount, network) => {
  if (network === "eip155:688689") {
    return {
      amount: (amount * 1e6).toString(), // USDC 6 decimals
      asset: usdcAddress,
      extra: {
        token: usdcName,
        name: usdcName,
        version: "2"
      }
    };
  }
  return null; // Use next parser
});

// Register EVM scheme
resourceServer.register(
  "eip155:688689",
  evmScheme
);

// Create Express application
const app = express();

// Middleware
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Configure payment middleware
app.use(
  paymentMiddleware(
    {
      "GET /data": {
        accepts: {
          scheme: "exact",
          price: "0.001",
          network: "eip155:688689",
          payTo: payToAddress,
        },
        description: "Random data service",
        mimeType: "application/json",
      },
      "GET /weather/:city": {
        accepts: {
          scheme: "exact",
          price: "0.002",
          network: "eip155:688689",
          payTo: payToAddress,
        },
        description: "Weather data service",
        mimeType: "application/json",
      },
      "POST /generate": {
        accepts: {
          scheme: "exact",
          price: "0.005",
          network: "eip155:688689",
          payTo: payToAddress,
        },
        description: "AI generation service",
        mimeType: "application/json",
      },
    },
    resourceServer
  )
);

// Health check endpoint (free)
app.get("/health", (req, res) => {
  res.json({
    status: "healthy",
    timestamp: new Date().toISOString(),
    network: "pharos-testnet",
    chainId: 688689,
    usdcAddress: usdcAddress,
    payToAddress: payToAddress,
  });
});

// Protected data endpoint
app.get("/data", (req, res) => {
  res.json({
    message: "Hello, paid user!",
    data: {
      timestamp: Date.now(),
      random: Math.random(),
      network: "pharos-testnet",
      chainId: 688689,
    },
    payment: {
      price: "0.001 USDC",
      network: "eip155:688689",
    },
  });
});

// Protected weather endpoint
app.get("/weather/:city", (req, res) => {
  const city = req.params.city || "Shanghai";
  
  // Simulate weather data
  const weatherData = {
    city,
    temperature: Math.floor(Math.random() * 30) + 10,
    condition: ["Sunny", "Cloudy", "Light Rain", "Overcast"][Math.floor(Math.random() * 4)],
    humidity: Math.floor(Math.random() * 40) + 40,
    windSpeed: Math.floor(Math.random() * 20) + 5,
    timestamp: new Date().toISOString(),
  };
  
  res.json({
    weather: weatherData,
    payment: {
      price: "0.002 USDC",
      network: "eip155:688689",
    },
  });
});

// Protected AI generation endpoint
app.post("/generate", (req, res) => {
  const { prompt, type = "text" } = req.body;
  
  if (!prompt) {
    return res.status(400).json({ error: "Prompt is required" });
  }
  
  // Simulate AI generation
  const generated = {
    id: Math.random().toString(36).substring(2, 15),
    type,
    prompt,
    result: `Generated ${type} for: ${prompt}`,
    tokens: Math.floor(Math.random() * 1000) + 100,
    timestamp: new Date().toISOString(),
  };
  
  res.json({
    generated,
    payment: {
      price: "0.005 USDC",
      network: "eip155:688689",
    },
  });
});

// Get server configuration
app.get("/config", (req, res) => {
  res.json({
    network: {
      name: "pharos-testnet",
      chainId: 688689,
      rpcUrl: "https://atlantic.dplabs-internal.com",
      usdcAddress: usdcAddress,
      facilitatorUrl: facilitatorUrl,
    },
    endpoints: [
      { path: "/health", price: "Free", description: "Health check" },
      { path: "/data", price: "0.001 USDC", description: "Random data" },
      { path: "/weather/:city", price: "0.002 USDC", description: "Weather data" },
      { path: "/generate", price: "0.005 USDC", description: "AI generation" },
    ],
    payToAddress: payToAddress,
  });
});

// Error handling
app.use((err: any, req: any, res: any, next: any) => {
  console.error('Error:', err);
  res.status(500).json({ 
    error: 'Internal server error',
    message: err.message 
  });
});

// 404 handling
app.use((req, res) => {
  res.status(404).json({ error: 'Endpoint not found' });
});

app.listen(port, () => {
  console.log(`Server running on http://localhost:${port}`);
});
```

### Client (Buyer) - Call Paid APIs

```typescript
// client.ts - Complete working implementation
import { wrapFetchWithPayment, x402Client, decodePaymentResponseHeader } from '@x402/fetch';
import { privateKeyToAccount } from 'viem/accounts';
import { config } from 'dotenv';
import fs from 'fs';
import { ExactEvmScheme } from '@x402/evm';

// Load environment variables
config();

// Securely get private key - read from environment variable or file
const privateKey = process.env.EVM_PRIVATE_KEY || 
  (fs.existsSync('.private_key') ? fs.readFileSync('.private_key', 'utf-8').trim() : null);

if (!privateKey) {
  console.error('Please set EVM_PRIVATE_KEY or create .private_key file');
  process.exit(1);
}

if (!privateKey.startsWith('0x')) {
  console.error('Private key must start with 0x');
  process.exit(1);
}

// Create signer
const signer = privateKeyToAccount(privateKey as `0x${string}`);

// Create x402 client
const client = new x402Client();

// Register EVM scheme
client.register("eip155:688689", new ExactEvmScheme(signer))

// Create fetch with payment
const fetchWithPayment = wrapFetchWithPayment(fetch, client);

// Usage example
async function main() {
  const serverUrl = process.argv[2];
  
  if (!serverUrl) {
    console.error('Please provide server URL');
    process.exit(1);
  }
  
  const response = await fetchWithPayment(serverUrl);
  const data = await response.json();
  console.log(data);
}

// Run main function
main().catch(console.error);
```

---

## package.json

```json
{
  "name": "x402-pharos-example",
  "version": "2.1.0",
  "type": "module",
  "scripts": {
    "server": "tsx server.ts",
    "client": "tsx client.ts http://localhost:4021/data",
    "dev:server": "tsx watch server.ts",
    "dev:client": "tsx watch client.ts"
  },
  "dependencies": {
    "@x402/core": "^2.0.0",
    "@x402/express": "^2.0.0",
    "@x402/fetch": "^2.0.0",
    "@x402/evm": "^2.0.0",
    "express": "^4.18.2",
    "viem": "^2.0.0",
    "dotenv": "^16.3.1"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "tsx": "^4.6.0",
    "typescript": "^5.3.0"
  }
}
```

---

## Dynamic Pricing Example

```typescript
// Server with dynamic pricing based on request parameters
{
  "GET /ai/:model": {
    accepts: (req) => ({
      scheme: "exact",
      price: req.params.model === "gpt4" ? "0.10" : "0.01",
      network: "eip155:688689",
      payTo: payToAddress,
    }),
  },
}
```

---

## Payment Flow

1. **Client** makes request to protected endpoint
2. **Server** returns HTTP 402 with `PAYMENT-REQUIRED` header
3. **Client** parses requirements, signs payment
4. **Client** re-sends request with `X-PAYMENT` header
5. **Server** forwards to **Facilitator** to verify
6. **Facilitator** settles payment on-chain (USDC)
7. **Server** returns response with `PAYMENT-RESPONSE` header

---

## Resources

- **x402 Docs**: https://docs.x402.org
- **Pharos Docs**: https://docs.pharosnetwork.io
- **x402 GitHub**: https://github.com/coinbase/x402