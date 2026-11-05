import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from './api.js';

// overdraft возвращает 500 из-за дефекта сервиса transactions: маппинг ошибки "недостаточно средств" в Internal
export const EXPECTED = {
  wrong_password: 401,
  foreign_balance: 404,
  bad_token: 401,
  invalid_body: 400,
  overdraft: 500,
};

const VARIANTS = ['wrong_password', 'foreign_balance', 'bad_token', 'invalid_body', 'overdraft'];

export function doErroneous(ctx) {
  const variant = VARIANTS[Math.floor(Math.random() * VARIANTS.length)];
  const expectedStatus = EXPECTED[variant];

  if (variant === 'wrong_password') {
    const res = http.post(
      `${ctx.baseUrl}/api/v1/auth/login`,
      JSON.stringify({
        email: ctx.company.email,
        password: 'IncorrectPassword123!',
      }),
      {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'POST /auth/login', kind: 'error', variant },
        responseCallback: http.expectedStatuses(expectedStatus),
      }
    );
    check(res, { 'wrong_password expected status': (r) => r.status === expectedStatus });
    return;
  }

  if (variant === 'foreign_balance') {
    const res = http.get(
      `${ctx.baseUrl}/api/v1/accounts/00000000-0000-0000-0000-000000000000/balance`,
      {
        headers: { Authorization: `Bearer ${ctx.token}` },
        tags: { name: 'GET /accounts/:id/balance', kind: 'error', variant },
        responseCallback: http.expectedStatuses(expectedStatus),
      }
    );
    check(res, { 'foreign_balance expected status': (r) => r.status === expectedStatus });
    return;
  }

  if (variant === 'bad_token') {
    const accountId = ctx.company.accounts[0];
    const res = http.get(
      `${ctx.baseUrl}/api/v1/accounts/${accountId}/balance`,
      {
        headers: { Authorization: 'Bearer garbage' },
        tags: { name: 'GET /accounts/:id/balance', kind: 'error', variant },
        responseCallback: http.expectedStatuses(expectedStatus),
      }
    );
    check(res, { 'bad_token expected status': (r) => r.status === expectedStatus });
    return;
  }

  if (variant === 'invalid_body') {
    const res = http.post(
      `${ctx.baseUrl}/api/v1/transactions/transfer`,
      JSON.stringify({
        from_account_id: 'not-a-uuid',
        to_account_id: ctx.company.accounts[1] || ctx.company.accounts[0],
        amount: 10,
        currency: 'RUB',
        idempotency_key: uuidv4(),
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${ctx.token}`,
        },
        tags: { name: 'POST /transactions/transfer', kind: 'error', variant },
        responseCallback: http.expectedStatuses(expectedStatus),
      }
    );
    check(res, { 'invalid_body expected status': (r) => r.status === expectedStatus });
    return;
  }

  if (variant === 'overdraft') {
    const res = http.post(
      `${ctx.baseUrl}/api/v1/transactions/transfer`,
      JSON.stringify({
        from_account_id: ctx.company.accounts[0],
        to_account_id: ctx.company.accounts[1] || ctx.company.accounts[0],
        amount: 1000000000000000,
        currency: 'RUB',
        idempotency_key: uuidv4(),
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${ctx.token}`,
        },
        tags: { name: 'POST /transactions/transfer', kind: 'error', variant },
        responseCallback: http.expectedStatuses(expectedStatus),
      }
    );
    check(res, { 'overdraft expected status': (r) => r.status === expectedStatus });
  }
}
