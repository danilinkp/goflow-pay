import { sleep } from 'k6';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';
import {
  login,
  logout,
  listAccounts,
  balance,
  history,
  transfer,
  statement,
  deposit,
  uuidv4,
} from './lib/api.js';
import { seedCompanies } from './lib/seed.js';
import { doErroneous } from './lib/errors.js';

const BASE_URL = __ENV.BASE_URL || 'http://gateway:8080';
const SCENARIO = __ENV.SCENARIO || 'smoke';
const ERROR_RATE = parseFloat(__ENV.ERROR_RATE || '0');

const SEED_COMPANIES = parseInt(__ENV.SEED_COMPANIES || '200', 10);
const THINK_MIN = parseFloat(__ENV.THINK_MIN || '1');
const THINK_MAX = parseFloat(__ENV.THINK_MAX || '3');
const ACTIONS_PER_SESSION = parseInt(__ENV.ACTIONS_PER_SESSION || '3', 10);

const RAMP_START_VUS = parseInt(__ENV.RAMP_START_VUS || '10', 10);
const RAMP_STEP_VUS = parseInt(__ENV.RAMP_STEP_VUS || '20', 10);
const RAMP_STEPS = parseInt(__ENV.RAMP_STEPS || '15', 10);
const RAMP_TRANSITION_DURATION = __ENV.RAMP_TRANSITION_DURATION || '10s';
const RAMP_STEP_DURATION = __ENV.RAMP_STEP_DURATION || '60s';
const RAMP_GRACEFUL_STOP = __ENV.RAMP_GRACEFUL_STOP || '30s';

const NORMAL_VUS = parseInt(__ENV.NORMAL_VUS || '50', 10);
const STRESS_VUS = parseInt(__ENV.STRESS_VUS || '300', 10);
const STRESS_BASELINE = __ENV.STRESS_BASELINE || '5m';
const STRESS_SPIKE_RAMP = __ENV.STRESS_SPIKE_RAMP || '30s';
const STRESS_HOLD = __ENV.STRESS_HOLD || '3m';
const STRESS_RECOVERY = __ENV.STRESS_RECOVERY || '10m';

const SOAK_VUS = parseInt(__ENV.SOAK_VUS || '50', 10);
const SOAK_DURATION = __ENV.SOAK_DURATION || '60m';

const SESSION_PAUSE_MIN = parseFloat(__ENV.SESSION_PAUSE_MIN || '2');
const SESSION_PAUSE_MAX = parseFloat(__ENV.SESSION_PAUSE_MAX || '5');

function buildStages(scenarioName) {
  if (scenarioName === 'smoke') {
    return [
      { duration: '5s', target: 5 },
      { duration: '55s', target: 5 },
    ];
  }

  if (scenarioName === 'ramp') {
    const stages = [];
    for (let s = 0; s < RAMP_STEPS; s++) {
      const target = RAMP_START_VUS + s * RAMP_STEP_VUS;
      stages.push({ duration: RAMP_TRANSITION_DURATION, target });
      stages.push({ duration: RAMP_STEP_DURATION, target });
    }
    return stages;
  }

  if (scenarioName === 'stress') {
    return [
      { duration: '10s', target: NORMAL_VUS },
      { duration: STRESS_BASELINE, target: NORMAL_VUS },
      { duration: STRESS_SPIKE_RAMP, target: STRESS_VUS },
      { duration: STRESS_HOLD, target: STRESS_VUS },
      { duration: '10s', target: NORMAL_VUS },
      { duration: STRESS_RECOVERY, target: NORMAL_VUS },
    ];
  }

  if (scenarioName === 'soak') {
    return [
      { duration: '30s', target: SOAK_VUS },
      { duration: SOAK_DURATION, target: SOAK_VUS },
    ];
  }

  return [
    { duration: '5s', target: 5 },
    { duration: '55s', target: 5 },
  ];
}

const scenarioConfig = {
  executor: 'ramping-vus',
  startVUs: SCENARIO === 'smoke' ? 5 : (SCENARIO === 'ramp' ? RAMP_START_VUS : NORMAL_VUS),
  stages: buildStages(SCENARIO),
  gracefulStop: SCENARIO === 'ramp' ? RAMP_GRACEFUL_STOP : '30s',
  gracefulRampDown: '30s',
};

const thresholds = {};
if (SCENARIO === 'ramp') {
  thresholds.http_req_failed = [
    { threshold: 'rate<0.5', abortOnFail: true, delayAbortEval: '1m' },
  ];
}

export const options = {
  setupTimeout: '15m',
  summaryTrendStats: ['min', 'avg', 'med', 'max', 'p(50)', 'p(75)', 'p(90)', 'p(95)', 'p(99)'],
  scenarios: {
    [SCENARIO]: scenarioConfig,
  },
  thresholds,
};

function uniform(min, max) {
  return min + Math.random() * (max - min);
}

function think() {
  sleep(uniform(THINK_MIN, THINK_MAX));
}

function executeWeightedAction(baseUrl, token, company, data) {
  const roll = Math.random() * 100;
  const ownAccounts = company.accounts;
  const accId = ownAccounts[Math.floor(Math.random() * ownAccounts.length)];

  if (roll < 40) {
    balance(baseUrl, token, accId);
    return;
  }

  if (roll < 65) {
    history(baseUrl, token, accId);
    return;
  }

  if (roll < 90) {
    let targetCompany = company;
    if (data.length > 1) {
      let idx = Math.floor(Math.random() * data.length);
      while (data[idx].email === company.email) {
        idx = Math.floor(Math.random() * data.length);
      }
      targetCompany = data[idx];
    }
    const toAccId = targetCompany.accounts[Math.floor(Math.random() * targetCompany.accounts.length)];
    const amount = Math.floor(Math.random() * 100) + 1;
    transfer(baseUrl, token, {
      from_account_id: accId,
      to_account_id: toAccId,
      amount,
      currency: 'RUB',
      idempotency_key: uuidv4(),
    });
    return;
  }

  if (roll < 95) {
    const now = new Date();
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    statement(baseUrl, token, accId, {
      period_from: yesterday.toISOString(),
      period_to: now.toISOString(),
    });
    return;
  }

  deposit(baseUrl, token, {
    account_id: accId,
    bank_account_id: company.bankAccountId,
    amount: 1000,
    idempotency_key: uuidv4(),
  });
}

export function setup() {
  const seedCount = SCENARIO === 'smoke' ? 10 : SEED_COMPANIES;
  return seedCompanies(BASE_URL, seedCount);
}

export default function (data) {
  if (!data || data.length === 0) {
    return;
  }

  const company = data[(__VU - 1) % data.length];
  const loginRes = login(BASE_URL, company.email, company.password);
  if (!loginRes || !loginRes.token) {
    return;
  }
  const token = loginRes.token;
  think();

  listAccounts(BASE_URL, token);
  think();

  for (let i = 0; i < ACTIONS_PER_SESSION; i++) {
    if (Math.random() < ERROR_RATE) {
      doErroneous({
        baseUrl: BASE_URL,
        token,
        company,
        data,
      });
    } else {
      executeWeightedAction(BASE_URL, token, company, data);
    }
    think();
  }

  logout(BASE_URL, token);
  sleep(uniform(SESSION_PAUSE_MIN, SESSION_PAUSE_MAX));
}

export function handleSummary(data) {
  return {
    '/results/summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
