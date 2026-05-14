import {apiProvider} from './client'
import type {
    Account,
    BankAccount,
    BankOperation,
    BankOperationRequest,
    CreateAccountRequest, GenerateStatementRequest,
    LinkBankAccountRequest, Statement,
} from '@/models/account'

export const accountsApi = {
    getAccounts: (companyId?: string) => {
        const query = companyId ? `?company_id=${companyId}` : ''
        return apiProvider.get<Account[]>(`/companies/accounts${query}`)
    },

    getBankAccounts: (companyId?: string) => {
        const query = companyId ? `?company_id=${companyId}` : ''
        return apiProvider.get<BankAccount[]>(`/companies/banks${query}`)
    },

    getBalance: (accountId: string) =>
        apiProvider.get<{ balance: number }>(`/accounts/${accountId}/balance`),

    createAccount: (req: CreateAccountRequest) =>
        apiProvider.post<Account>('/accounts', req),

    deactivateAccount: (accountId: string, companyId?: string) => {
        const query = companyId ? `?company_id=${companyId}` : ''
        return apiProvider.delete<Account>(`/accounts/${accountId}${query}`)
    },


    linkBankAccount: (req: LinkBankAccountRequest) =>
        apiProvider.post<BankAccount>('/accounts/bank', req),

    bankDeposit: (req: BankOperationRequest) =>
        apiProvider.post<BankOperation>('/accounts/bank/deposit', req),

    bankWithdrawal: (req: BankOperationRequest) =>
        apiProvider.post<BankOperation>('/accounts/bank/withdrawal', req),

    generateStatement: (accountId: string, req: GenerateStatementRequest) =>
        apiProvider.post<Statement>(`/accounts/${accountId}/statement`, req),
}