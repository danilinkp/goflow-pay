import {apiProvider} from './client'
import type { Transaction, TransferRequest } from '@/models/transaction'

export const transactionsApi = {
    getTransactions: (accountId: string) =>
        apiProvider.get<Transaction[]>(`/transactions/${accountId}`),

    getTransaction: (txId: string) =>
        apiProvider.get<Transaction>(`/transactions/detail/${txId}`),

    transfer: (req: TransferRequest) =>
        apiProvider.post<Transaction>('/transactions/transfer', req),
}