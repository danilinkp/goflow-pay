import { register, createAccount, linkBank, deposit, uuidv4 } from './api.js';

const WEIGHTS = [7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1];

function generateSettlementAccount(bic, i) {
  const prefix = '40702643';
  const suffix = String(i).padStart(11, '0');
  const bicSuffix = bic.slice(6);
  for (let d = 0; d < 10; d++) {
    const acc = `${prefix}${d}${suffix}`;
    const key = bicSuffix + acc;
    let s = 0;
    for (let j = 0; j < 23; j++) {
      s += (parseInt(key[j], 10) * WEIGHTS[j]) % 10;
    }
    if (s % 10 === 0) {
      return acc;
    }
  }
  return '40702643000000000009';
}

export function seedCompanies(baseUrl, n) {
  const runId = uuidv4().slice(0, 8);
  const password = 'Password123!';
  const companies = [];

  for (let i = 1; i <= n; i++) {
    const login = `load_${i}_${runId}`;
    const email = `${login}@example.com`;
    const companyName = `Company ${login}`;

    const reg = register(
      baseUrl,
      {
        login,
        email,
        password,
        company_name: companyName,
      },
      { phase: 'setup' }
    );
    if (!reg || !reg.token) {
      throw new Error(`Failed to register seed company ${login}`);
    }

    const token = reg.token;
    const acc1 = createAccount(baseUrl, token, 'RUB', { phase: 'setup' });
    const acc2 = createAccount(baseUrl, token, 'RUB', { phase: 'setup' });
    if (!acc1 || !acc1.account_id || !acc2 || !acc2.account_id) {
      throw new Error(`Failed to create accounts for seed company ${login}`);
    }

    const bank = linkBank(
      baseUrl,
      token,
      {
        account_id: acc1.account_id,
        name: `Bank ${i}`,
        bic: '044525225',
        settlement_account: generateSettlementAccount('044525225', i),
        currency: 'RUB',
      },
      { phase: 'setup' }
    );
    if (!bank || !bank.bank_account_id) {
      throw new Error(`Failed to link bank account for seed company ${login}`);
    }

    deposit(
      baseUrl,
      token,
      {
        account_id: acc1.account_id,
        bank_account_id: bank.bank_account_id,
        amount: 10000000,
        idempotency_key: uuidv4(),
      },
      { phase: 'setup' }
    );

    deposit(
      baseUrl,
      token,
      {
        account_id: acc2.account_id,
        bank_account_id: bank.bank_account_id,
        amount: 10000000,
        idempotency_key: uuidv4(),
      },
      { phase: 'setup' }
    );

    companies.push({
      email,
      password,
      accounts: [acc1.account_id, acc2.account_id],
      bankAccountId: bank.bank_account_id,
    });
  }

  return companies;
}
