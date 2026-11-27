import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Gauge, Rate, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://nginx';
const SCENARIO = __ENV.SCENARIO || 'steady';
const READ_RPS = Number(__ENV.READ_RPS || 150);
const WRITE_RPS = Number(__ENV.WRITE_RPS || 15);
const DURATION = __ENV.DURATION || '2m';
const RAMP_PEAK = Number(__ENV.RAMP_PEAK || 800);
const COMPANIES = Number(__ENV.COMPANIES || 5);

const INSTANCES = {
  [__ENV.GATEWAY_ADDR]: 'gateway',
  [__ENV.GATEWAY_RO_1_ADDR]: 'gateway-ro-1',
  [__ENV.GATEWAY_RO_2_ADDR]: 'gateway-ro-2',
};

const readsGateway = new Counter('reads_gateway');
const readsRo1 = new Counter('reads_gateway_ro_1');
const readsRo2 = new Counter('reads_gateway_ro_2');
const readsUnknown = new Counter('reads_unknown');
const readOnPrimary = new Rate('read_on_primary');
const writeOnPrimary = new Rate('write_on_primary');
const rateLimited = new Rate('rate_limited');
const runStarted = new Gauge('run_started');
const runFinished = new Gauge('run_finished');
const readLatency = new Trend('read_latency', true);
const writeLatency = new Trend('write_latency', true);

const READ_COUNTERS = {
  gateway: readsGateway,
  'gateway-ro-1': readsRo1,
  'gateway-ro-2': readsRo2,
};

const SCENARIOS = {
  steady: {
    reads: {
      executor: 'constant-arrival-rate',
      exec: 'read',
      rate: READ_RPS,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 300,
    },
  },
  mixed: {
    reads: {
      executor: 'constant-arrival-rate',
      exec: 'read',
      rate: READ_RPS,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 50,
      maxVUs: 300,
    },
    writes: {
      executor: 'constant-arrival-rate',
      exec: 'write',
      rate: WRITE_RPS,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 10,
      maxVUs: 100,
    },
  },
  ramp: {
    reads: {
      executor: 'ramping-arrival-rate',
      exec: 'read',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 1000,
      stages: [
        { target: Math.round(RAMP_PEAK * 0.25), duration: '30s' },
        { target: Math.round(RAMP_PEAK * 0.25), duration: '30s' },
        { target: Math.round(RAMP_PEAK * 0.5), duration: '30s' },
        { target: Math.round(RAMP_PEAK * 0.5), duration: '30s' },
        { target: Math.round(RAMP_PEAK * 0.75), duration: '30s' },
        { target: Math.round(RAMP_PEAK * 0.75), duration: '30s' },
        { target: RAMP_PEAK, duration: '30s' },
        { target: RAMP_PEAK, duration: '30s' },
      ],
    },
  },
  ratelimit: {
    burst: {
      executor: 'constant-arrival-rate',
      exec: 'read',
      rate: 50,
      timeUnit: '1s',
      duration: '20s',
      preAllocatedVUs: 20,
      maxVUs: 50,
    },
  },
};

const THRESHOLDS = {
  steady: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{kind:read}': ['p(95)<300'],
    read_on_primary: ['rate>0.48', 'rate<0.52'],
    reads_unknown: ['count==0'],
  },
  mixed: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{kind:read}': ['p(95)<300'],
    'http_req_duration{kind:write}': ['p(95)<500'],
    read_on_primary: ['rate>0.48', 'rate<0.52'],
    write_on_primary: ['rate==1'],
    reads_unknown: ['count==0'],
  },
  ramp: {
    read_on_primary: ['rate>0.48', 'rate<0.52'],
  },
  ratelimit: {
    rate_limited: ['rate>0.5'],
  },
};

export const options = {
  scenarios: SCENARIOS[SCENARIO],
  thresholds: THRESHOLDS[SCENARIO],
  setupTimeout: '2m',
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function headers(token, body) {
  const h = { Authorization: `Bearer ${token}` };
  if (body) {
    h['Content-Type'] = 'application/json';
  }
  return h;
}

function instanceOf(res) {
  return INSTANCES[res.headers['X-Upstream']] || 'unknown';
}

function post(path, payload, token) {
  return http.post(`${BASE_URL}${path}`, JSON.stringify(payload), {
    headers: token ? headers(token, true) : { 'Content-Type': 'application/json' },
    tags: { kind: 'setup' },
  });
}

function retryPost(path, payload, token, expected) {
  for (let attempt = 0; attempt < 20; attempt++) {
    const res = post(path, payload, token);
    if (res.status === expected) {
      return res.json();
    }
    if (res.status !== 429) {
      throw new Error(`${path}: ${res.status} ${res.body}`);
    }
    sleep(1);
  }
  throw new Error(`${path}: rate limited`);
}

export function setup() {
  runStarted.add(Date.now() / 1000);
  const runId = Math.random().toString(36).slice(2, 8);
  const companies = [];
  for (let i = 0; i < COMPANIES; i++) {
    const login = `web_load_${runId}_${i}`;
    const reg = retryPost(
      '/api/v1/auth/register/company',
      { login, email: `${login}@example.com`, password: 'Password123!', company_name: `Load ${login}` },
      null,
      201,
    );
    const accounts = [];
    for (let j = 0; j < 2; j++) {
      accounts.push(retryPost('/api/v1/accounts', { currency: 'RUB' }, reg.token, 201).account_id);
    }
    companies.push({ token: reg.token, accounts });
  }
  return { companies };
}

export function teardown() {
  runFinished.add(Date.now() / 1000);
}

const READS = [
  { name: 'GET /api/v1/companies/accounts', path: () => '/api/v1/companies/accounts' },
  { name: 'GET /api/v1/accounts/:id/balance', path: (c) => `/api/v1/accounts/${c.accounts[0]}/balance` },
  { name: 'GET /api/v2/accounts', path: () => '/api/v2/accounts' },
  { name: 'GET /api/v2/accounts/:id', path: (c) => `/api/v2/accounts/${c.accounts[1]}` },
  { name: 'GET /api/v2/auth/me', path: () => '/api/v2/auth/me' },
];

function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

export function read(data) {
  const company = pick(data.companies);
  const op = pick(READS);
  const res = http.get(`${BASE_URL}${op.path(company)}`, {
    headers: headers(company.token, false),
    tags: { name: op.name, kind: 'read' },
    responseCallback: http.expectedStatuses(200),
  });

  if (SCENARIO === 'ratelimit') {
    rateLimited.add(res.status === 429);
    check(res, { 'allowed or limited': (r) => r.status === 200 || r.status === 429 });
    return;
  }

  check(res, { 'read 200': (r) => r.status === 200 });
  readLatency.add(res.timings.duration);
  if (res.status !== 200) {
    return;
  }
  const instance = instanceOf(res);
  readOnPrimary.add(instance === 'gateway');
  (READ_COUNTERS[instance] || readsUnknown).add(1);
}

export function write(data) {
  const company = pick(data.companies);
  const res = http.post(`${BASE_URL}/api/v1/accounts`, JSON.stringify({ currency: 'RUB' }), {
    headers: headers(company.token, true),
    tags: { name: 'POST /api/v1/accounts', kind: 'write' },
    responseCallback: http.expectedStatuses(201),
  });
  check(res, { 'write 201': (r) => r.status === 201 });
  writeLatency.add(res.timings.duration);
  writeOnPrimary.add(instanceOf(res) === 'gateway');
}
