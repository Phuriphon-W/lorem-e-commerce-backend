import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { BASE_URL, registerUser, getAuthHeaders } from './config.js';

// Custom metrics to verify reads vs. writes separately
const readDurationTrend = new Trend('read_duration');
const writeDurationTrend = new Trend('write_duration');

export const options = {
  stages: [
    { duration: '30s', target: 1000 }, // Ramp up to 1000 VUs
    { duration: '2m', target: 1000 },   // Sustain 1000 VUs for 2 minutes
    { duration: '30s', target: 0 },    // Ramp down to 0 VUs
  ],
  thresholds: {
    http_req_failed: ['rate<0.02'],       // Overall success rate >= 98%
    'http_req_failed{status:500}': ['rate<0.005'], // HTTP 500 < 0.5%
    read_duration: ['p(95)<500'],          // p95 reads < 500ms
    write_duration: ['p(95)<2000'],        // p95 writes < 2s
  },
};

// Setup: fetch categories and product IDs to use during the test
export function setup() {
  const categoriesRes = http.get(`${BASE_URL}/api/category`);
  const categories = categoriesRes.json().categories || [];

  const productsRes = http.get(`${BASE_URL}/api/product?pageSize=20`);
  const products = productsRes.json().products || [];

  return {
    categories: categories.map((c) => c.name),
    productIds: products.map((p) => p.id),
  };
}

let session = null;

export default function (data) {
  const { categories, productIds } = data;
  
  // Decide which action to take based on the target weights
  const rand = Math.random();

  if (rand < 0.50) {
    // 1. Browse products (50%)
    const page = Math.floor(Math.random() * 2) + 1;
    const res = http.get(`${BASE_URL}/api/product?pageNumber=${page}&pageSize=10`);
    readDurationTrend.add(res.timings.duration);
    check(res, { 'browse status is 200': (r) => r.status === 200 });
  } 
  else if (rand < 0.70) {
    // 2. View product detail (20%)
    if (productIds && productIds.length > 0) {
      const pid = productIds[Math.floor(Math.random() * productIds.length)];
      const res = http.get(`${BASE_URL}/api/product/${pid}`);
      readDurationTrend.add(res.timings.duration);
      check(res, { 'product detail status is 200': (r) => r.status === 200 });
    } else {
      sleep(0.5);
    }
  } 
  else if (rand < 0.80) {
    // 3. View categories (10%)
    const res = http.get(`${BASE_URL}/api/category`);
    readDurationTrend.add(res.timings.duration);
    check(res, { 'category list status is 200': (r) => r.status === 200 });
  } 
  else {
    // Write actions require authentication
    if (!session) {
      session = registerUser(__VU);
      if (!session) {
        sleep(1);
        return;
      }
    }

    const authHeaders = getAuthHeaders(session.authToken);

    if (rand < 0.90) {
      // 4. Add to cart (10%)
      if (productIds && productIds.length > 0) {
        const pid = productIds[Math.floor(Math.random() * productIds.length)];
        const payload = JSON.stringify({
          productId: pid,
          quantity: 1,
        });
        const res = http.post(`${BASE_URL}/api/user/${session.userId}/cart`, payload, authHeaders);
        writeDurationTrend.add(res.timings.duration);
        check(res, { 'add to cart returns 2xx or 400': (r) => r.status === 200 || r.status === 201 || r.status === 400 });
      } else {
        sleep(0.5);
      }
    } 
    else if (rand < 0.95) {
      // 5. Create order (5%)
      if (productIds && productIds.length > 0) {
        const pid = productIds[Math.floor(Math.random() * productIds.length)];
        const payload = JSON.stringify({
          userId: session.userId,
          items: [{ productId: pid, quantity: 1 }],
        });
        const res = http.post(`${BASE_URL}/api/order`, payload, authHeaders);
        writeDurationTrend.add(res.timings.duration);
        check(res, { 'order returns 2xx or 400': (r) => r.status === 200 || r.status === 201 || r.status === 400 });
      } else {
        sleep(0.5);
      }
    } 
    else {
      // 6. Checkout (5%)
      // First make a quick order, then checkout
      if (productIds && productIds.length > 0) {
        const pid = productIds[Math.floor(Math.random() * productIds.length)];
        const orderPayload = JSON.stringify({
          userId: session.userId,
          items: [{ productId: pid, quantity: 1 }],
        });
        const orderRes = http.post(`${BASE_URL}/api/order`, orderPayload, authHeaders);
        writeDurationTrend.add(orderRes.timings.duration);

        if (orderRes.status === 200 || orderRes.status === 201) {
          const orderId = orderRes.json().id;
          const checkoutPayload = JSON.stringify({
            userId: session.userId,
            orderId: orderId,
          });
          const checkoutRes = http.post(`${BASE_URL}/api/payment/checkout`, checkoutPayload, authHeaders);
          writeDurationTrend.add(checkoutRes.timings.duration);
          check(checkoutRes, { 'checkout returns 2xx': (r) => r.status === 200 || r.status === 201 });
        }
      } else {
        sleep(0.5);
      }
    }
  }

  sleep(Math.random() * 2 + 1); // Think time: 1-3 seconds
}
