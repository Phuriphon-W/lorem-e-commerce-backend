# k6 Backend Load Testing Suite

This directory contains a suite of load testing scripts implemented with [k6](https://k6.io/) to validate the backend's stability, latency, and breaking limits.

## Scripts Overview

1. **`config.js`**: Shared variables, base URL config, and helpers for registering and authenticating virtual users.
2. **`product-listing.js`**: Public read-heavy endpoint scenario. Simulates up to 1000 concurrent VUs searching, paginating, and sorting products.
3. **`order-creation.js`**: Concurrency stress scenario. 500 VUs simultaneously compete to order items from a shared pool of 10 products, forcing row locks.
4. **`file-upload.js`**: Admin I/O-bound scenario. 200 VUs concurrently upload 100KB files to check S3 upload throughput and PostgreSQL metadata inserts.
5. **`auth-burst.js`**: Rate limit test. Verifies that single-IP bursts get throttled (with HTTP 429) while multi-IP concurrent traffic is allowed.
6. **`mixed-workload.js`**: A realistic distribution modeling overall application traffic (50% browse, 20% product details, 10% categories, 10% cart additions, 5% orders, 5% checkouts).

## Prerequisites

1. **k6**: Ensure `k6` is installed on your local host machine.
   - [k6 Installation Instructions](https://k6.io/docs/getting-started/installation/)
2. **Backend Services**: Ensure the PostgreSQL, Redis, and S3-compatible service (e.g. LocalStack or AWS S3) are running.
   - Run `make dev-up` or `make prod-up` from the root directory.
3. **Seed Database**: The database must have seeded data (at least 10 products and users).
   - Run `make seed-db` from the root directory.

## How to Run the Tests

You can run individual scripts or execute the entire suite sequentially.

### Run a Single Script
To run a specific test script:
```bash
# From the backend directory
k6 run loadtest/product-listing.js

# Override the Base URL
k6 run -e BASE_URL=http://localhost:5000 loadtest/product-listing.js
```

### Run the Whole Suite
Use the Makefile target to run all tests sequentially:
```bash
make loadtest
```

## Interpreting Results

- **Pass Criteria**:
  - HTTP Request Success Rate: Look at `http_req_failed` or the custom scenario checks.
  - Latency: Look at `http_req_duration` (specifically the `p95` and `p99` percentiles).
- **Error Logs**: Look for any HTTP 500 status codes. If you see HTTP 500s during `order-creation.js`, it signifies connection pool starvation or unhandled deadlock exceptions in Go.
