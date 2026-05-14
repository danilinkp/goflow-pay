export interface Account {
    account_id: string
    company_id: string
    balance: number
    currency: string
    status: string
    created_at: string
}

export interface BankAccount {
    bank_account_id: string
    company_id: string
    name: string
    bic: string
    settlement_account: string
    currency: string
    created_at: string
}

export interface BankOperation {
    bank_operation_id: string
    account_id: string
    bank_account_id: string
    operation_type: string
    operation_status: string
    amount: number
    idempotency_key: string
    external_id: string
    created_at: string
}

export interface CreateAccountRequest {
    currency: string
}

export interface LinkBankAccountRequest {
    account_id: string
    name: string
    bic: string
    settlement_account: string
    currency: string
}

export interface BankOperationRequest {
    account_id: string
    bank_account_id: string
    amount: number
    idempotency_key: string
}

export interface StatementEntry {
    date: string
    entry_type: string
    amount: number
    balance_after: number
    counterparty: string
}

export interface Statement {
    statement_id: string
    account_id: string
    company_id: string
    initiator_id: string
    period_from: string
    period_to: string
    opening_balance: number
    closing_balance: number
    total_debit: number
    total_credit: number
    currency: string
    entries: StatementEntry[]
    created_at: string
}

export interface GenerateStatementRequest {
    period_from: string
    period_to: string
}