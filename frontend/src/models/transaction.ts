export interface Transaction {
    transaction_id: string
    initiator_id: string
    from_account_id: string
    to_account_id: string
    amount: number
    currency: string
    idempotency_key: string
    transaction_status: string
    created_at: string
}

export interface TransferRequest {
    from_account_id: string
    to_account_id: string
    amount: number
    currency: string
    idempotency_key: string
}