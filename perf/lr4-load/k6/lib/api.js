import http from 'k6/http';
import { check } from 'k6';

export function uuidv4() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

function buildHeaders(token, hasBody = true) {
  const headers = {};
  if (hasBody) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

export function register(baseUrl, data, extraTags = {}) {
  const url = `${baseUrl}/api/v1/auth/register/company`;
  const params = {
    headers: buildHeaders(null, true),
    tags: Object.assign({ name: 'POST /auth/register/company', kind: 'normal' }, extraTags),
  };
  const res = http.post(url, JSON.stringify(data), params);
  const ok = check(res, { 'register status 201': (r) => r.status === 201 });
  if (!ok) {
    return null;
  }
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}

export function login(baseUrl, email, password, extraTags = {}) {
  const url = `${baseUrl}/api/v1/auth/login`;
  const params = {
    headers: buildHeaders(null, true),
    tags: Object.assign({ name: 'POST /auth/login', kind: 'normal' }, extraTags),
  };
  const res = http.post(url, JSON.stringify({ email, password }), params);
  const ok = check(res, { 'login status 200': (r) => r.status === 200 });
  if (!ok) {
    return null;
  }
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}

export function logout(baseUrl, token, extraTags = {}) {
  const url = `${baseUrl}/api/v1/auth/logout`;
  const params = {
    headers: buildHeaders(token, false),
    tags: Object.assign({ name: 'POST /auth/logout', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.post(url, null, params);
  const ok = check(res, { 'logout status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function createAccount(baseUrl, token, currency = 'RUB', extraTags = {}) {
  const url = `${baseUrl}/api/v1/accounts`;
  const params = {
    headers: buildHeaders(token, true),
    tags: Object.assign({ name: 'POST /accounts', kind: 'normal' }, extraTags),
  };
  const res = http.post(url, JSON.stringify({ currency }), params);
  const ok = check(res, { 'createAccount status 201': (r) => r.status === 201 });
  if (!ok) {
    return null;
  }
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}

export function linkBank(baseUrl, token, data, extraTags = {}) {
  const url = `${baseUrl}/api/v1/accounts/bank`;
  const params = {
    headers: buildHeaders(token, true),
    tags: Object.assign({ name: 'POST /accounts/bank', kind: 'normal' }, extraTags),
  };
  const res = http.post(url, JSON.stringify(data), params);
  const ok = check(res, { 'linkBank status 201': (r) => r.status === 201 });
  if (!ok) {
    return null;
  }
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}

export function deposit(baseUrl, token, data, extraTags = {}) {
  const url = `${baseUrl}/api/v1/accounts/bank/deposit`;
  const params = {
    headers: buildHeaders(token, true),
    tags: Object.assign({ name: 'POST /accounts/bank/deposit', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.post(url, JSON.stringify(data), params);
  const ok = check(res, { 'deposit status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function listAccounts(baseUrl, token, extraTags = {}) {
  const url = `${baseUrl}/api/v1/companies/accounts`;
  const params = {
    headers: buildHeaders(token, false),
    tags: Object.assign({ name: 'GET /companies/accounts', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.get(url, params);
  const ok = check(res, { 'listAccounts status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function balance(baseUrl, token, accountId, extraTags = {}) {
  const url = `${baseUrl}/api/v1/accounts/${accountId}/balance`;
  const params = {
    headers: buildHeaders(token, false),
    tags: Object.assign({ name: 'GET /accounts/:id/balance', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.get(url, params);
  const ok = check(res, { 'balance status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function history(baseUrl, token, accountId, extraTags = {}) {
  const url = `${baseUrl}/api/v1/transactions/${accountId}`;
  const params = {
    headers: buildHeaders(token, false),
    tags: Object.assign({ name: 'GET /transactions/:id', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.get(url, params);
  const ok = check(res, { 'history status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function transfer(baseUrl, token, data, extraTags = {}) {
  const url = `${baseUrl}/api/v1/transactions/transfer`;
  const params = {
    headers: buildHeaders(token, true),
    tags: Object.assign({ name: 'POST /transactions/transfer', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.post(url, JSON.stringify(data), params);
  const ok = check(res, { 'transfer status 200': (r) => r.status === 200 });
  return ok ? true : null;
}

export function statement(baseUrl, token, accountId, data, extraTags = {}) {
  const url = `${baseUrl}/api/v1/accounts/${accountId}/statement`;
  const params = {
    headers: buildHeaders(token, true),
    tags: Object.assign({ name: 'POST /accounts/:id/statement', kind: 'normal' }, extraTags),
    responseType: 'none',
  };
  const res = http.post(url, JSON.stringify(data), params);
  const ok = check(res, { 'statement status 200': (r) => r.status === 200 });
  return ok ? true : null;
}
