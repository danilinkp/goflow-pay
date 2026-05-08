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