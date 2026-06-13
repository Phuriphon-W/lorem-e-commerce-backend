import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL } from './config.js';

export const options = {
  stages: [
    { duration: '30s', target: 1000 },  // Ramp up to 1000 VUs
    { duration: '2m', target: 1000 },   // Sustain 1000 VUs
    { duration: '15s', target: 0 },     // Ramp down to 0 VUs
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],    // <1% errors
    http_req_duration: ['p(95)<500', 'p(99)<2000'], // p95 < 500ms, p99 < 2s
  },
};

const CATEGORIES = ['Apparel', 'Accessory', ''];
const SEARCH_KEYWORDS = ['Cotton', 'Leather', 'Classic', 'Sweater', 'Watch', ''];
const ORDER_OPTIONS = ['price_low', 'price_high', 'name_asc', 'name_desc', 'date_asc', ''];

export default function () {
  const category = CATEGORIES[Math.floor(Math.random() * CATEGORIES.length)];
  const search = SEARCH_KEYWORDS[Math.floor(Math.random() * SEARCH_KEYWORDS.length)];
  const orderBy = ORDER_OPTIONS[Math.floor(Math.random() * ORDER_OPTIONS.length)];
  const pageNumber = Math.floor(Math.random() * 2) + 1; // page 1 or 2
  const pageSize = 20;

  let url = `${BASE_URL}/api/product?pageNumber=${pageNumber}&pageSize=${pageSize}`;
  if (category) url += `&category=${encodeURIComponent(category)}`;
  if (search) url += `&search=${encodeURIComponent(search)}`;
  if (orderBy) url += `&orderBy=${orderBy}`;

  const res = http.get(url);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has products list': (r) => r.json() && Array.isArray(r.json().products),
  });

  sleep(Math.random() * 2 + 1); // Think time: 1-3 seconds
}
