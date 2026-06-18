import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, signinUser } from './config.js';

export const options = {
  stages: [
    { duration: '15s', target: 200 },
    { duration: '45s', target: 200 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<3000'],
  },
};

const fileData = 'x'.repeat(100 * 1024); // 100KB of dummy file content

export function setup() {
  const adminSession = signinUser('admin@example.com', 'password123');
  return {
    adminToken: adminSession ? adminSession.authToken : null,
  };
}

export default function (data) {
  if (!data.adminToken) {
    sleep(1);
    return;
  }

  const payload = {
    file: http.file(fileData, 'test-image.jpg', 'image/jpeg'),
    objectBaseKey: 'test-uploads',
  };

  const params = {
    headers: {
      'Cookie': `authToken=${data.adminToken}`,
    },
  };

  const res = http.post(`${BASE_URL}/api/file/upload`, payload, params);

  check(res, {
    'status is 201': (r) => r.status === 201,
    'has fileId': (r) => r.json() && r.json().fileId !== undefined,
  });

  sleep(Math.random() * 2 + 1);
}
