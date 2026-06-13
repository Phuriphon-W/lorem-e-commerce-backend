import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, signinUser, getAuthHeaders } from './config.js';

export const options = {
  scenarios: {
    order_contention: {
      executor: 'per-vu-iterations',
      vus: 500,
      iterations: 3,
      maxDuration: '1m',
    },
  },
  thresholds: {
    // 500 Internal Server Errors must be 0
    'http_req_failed{status:500}': ['rate==0'],
    http_req_duration: ['p(95)<2000'],
  },
};

export function setup() {
  const res = http.get(`${BASE_URL}/api/product?pageSize=10`);
  const products = res.json().products || [];
  const productIds = products.map((p) => p.id);

  if (productIds.length === 0) {
    console.log('WARNING: No products found. Seed the database first.');
  }

  // Sign in once as the seeded customer user (uses unique IP to bypass initial limit)
  const session = signinUser('testuser@example.com', 'password123', 999);

  return {
    productIds,
    authToken: session ? session.authToken : null,
    userId: session ? session.userId : null,
  };
}

export default function (data) {
  const { productIds, authToken, userId } = data;
  if (!productIds || productIds.length === 0 || !authToken || !userId) {
    sleep(1);
    return;
  }

  // Choose 1-3 random products from the set of 10
  const numItems = Math.floor(Math.random() * 3) + 1;
  const items = [];
  const chosenIndices = new Set();

  while (chosenIndices.size < numItems) {
    const idx = Math.floor(Math.random() * productIds.length);
    chosenIndices.add(idx);
  }

  for (const idx of chosenIndices) {
    items.push({
      productId: productIds[idx],
      quantity: Math.floor(Math.random() * 2) + 1, // 1 or 2 items
    });
  }

  const payload = JSON.stringify({
    userId: userId,
    items: items,
  });

  const params = getAuthHeaders(authToken);
  const res = http.post(`${BASE_URL}/api/order`, payload, params);

  check(res, {
    'is not 500': (r) => r.status !== 500,
    'order created or out of stock': (r) => r.status === 200 || r.status === 201 || r.status === 400,
  });

  sleep(Math.random() * 2 + 1); // Think time: 1-3 seconds
}
