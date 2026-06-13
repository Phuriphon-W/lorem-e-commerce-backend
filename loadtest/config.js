import http from 'k6/http';
import { check } from 'k6';

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:5000';

// Default password for all test users
export const TEST_PASSWORD = 'password123';

/**
 * Register a new test user and return their authentication cookie
 */
export function registerUser(vuId) {
  const username = `testvu_${vuId}_${Date.now()}`.substring(0, 20); // ensure within maxLength 20
  const payload = JSON.stringify({
    username: username,
    firstName: 'Test',
    lastName: 'VU',
    email: `${username}@example.com`,
    password: TEST_PASSWORD,
  });

  const ip = `192.168.1.${vuId}`;
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Forwarded-For': ip,
      'X-Real-IP': ip,
    },
  };

  const res = http.post(`${BASE_URL}/auth/register`, payload, params);
  const success = check(res, {
    'register status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    return null;
  }

  // Extract authToken from cookies
  const cookies = res.cookies;
  const authToken = cookies['authToken'] ? cookies['authToken'][0].value : null;
  const userId = res.json() ? res.json().id : null;

  return { authToken, userId, email: `${username}@example.com` };
}

/**
 * Sign in a user and return their authentication cookie
 */
export function signinUser(email, password = TEST_PASSWORD, vuId = 1) {
  const payload = JSON.stringify({
    email: email,
    password: password,
  });

  const ip = `192.168.2.${vuId}`;
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Forwarded-For': ip,
      'X-Real-IP': ip,
    },
  };

  const res = http.post(`${BASE_URL}/auth/signin`, payload, params);
  const success = check(res, {
    'signin status is 200': (r) => r.status === 200,
  });

  if (!success) {
    return null;
  }

  const cookies = res.cookies;
  const authToken = cookies['authToken'] ? cookies['authToken'][0].value : null;
  const userId = res.json() ? res.json().id : null;

  return { authToken, userId };
}

/**
 * Get standard headers for authorized requests
 */
export function getAuthHeaders(token) {
  return {
    headers: {
      'Cookie': `authToken=${token}`,
      'Content-Type': 'application/json',
    },
  };
}
