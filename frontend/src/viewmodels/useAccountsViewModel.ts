import {useCallback, useEffect, useMemo, useState} from 'react'
import {accountsApi} from '@/api/accounts'
import {useAuthStore} from '@/store/authStore'
import type {Account, BankAccount, Statement} from '@/models/account'
import {AxiosError} from "axios"
import {mapTechnicalError} from "@/utils/baseError.ts"; // Не забываем импорт

export function useAccountsViewModel() {
    const {role} = useAuthStore()
    const [accounts, setAccounts] = useState<Account[]>([])
    const [bankAccounts, setBankAccounts] = useState<BankAccount[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [success, setSuccess] = useState<string | null>(null)

    const handleAction = async (action: () => Promise<void>, defaultError: string) => {
        setLoading(true)
        setError(null)
        setSuccess(null)
        try {
            await action()
            return true
        } catch (e) {
            setError(mapAccountsError(e, defaultError))
            return false
        } finally {
            setLoading(false)
        }
    }

    const fetchAccounts = useCallback(async (companyId?: string) => {
        await handleAction(async () => {
            const [accs, banks] = await Promise.all([
                accountsApi.getAccounts(companyId),
                accountsApi.getBankAccounts(companyId),
            ])
            setAccounts(accs)
            setBankAccounts(banks)
        }, 'Не удалось загрузить счета')
    }, [])

    const activeAccounts = useMemo(() => {
        return accounts.filter(acc => acc.status !== 'ACCOUNT_STATUS_INACTIVE');
    }, [accounts]);

    useEffect(() => {
        fetchAccounts()
    }, [fetchAccounts])

    const createAccount = async (currency: string) => {
        return await handleAction(async () => {
            const acc = await accountsApi.createAccount({currency})
            setAccounts((prev) => [...prev, acc])
            setSuccess('Счет успешно создан')
        }, 'Ошибка при создании счета')
    }

    const deactivateAccount = async (accountId: string, companyId?: string) => {
        await handleAction(async () => {
            await accountsApi.deactivateAccount(accountId, companyId)
            setAccounts((prev) =>
                prev.map((a) =>
                    a.account_id === accountId ? {...a, status: 'ACCOUNT_STATUS_INACTIVE'} : a
                )
            )
            setSuccess('Счет деактивирован')
        }, 'Не удалось деактивировать счет')
    }

    const linkBankAccount = async (
        accountId: string,
        name: string,
        bic: string,
        settlementAccount: string,
        currency: string
    ) => {
        return await handleAction(async () => {
            const ba = await accountsApi.linkBankAccount({
                account_id: accountId,
                name,
                bic,
                settlement_account: settlementAccount,
                currency,
            })
            setBankAccounts((prev) => [...prev, ba])
            setSuccess('Банковский счет привязан')
        }, 'Ошибка при привязке счета')
    }

    const bankDeposit = async (
        accountId: string,
        bankAccountId: string,
        amount: number,
        idempotencyKey: string
    ) => {
        return await handleAction(async () => {
            await accountsApi.bankDeposit({
                account_id: accountId,
                bank_account_id: bankAccountId,
                amount,
                idempotency_key: idempotencyKey
            })
            setSuccess('Запрос на пополнение отправлен')
            await fetchAccounts()
        }, 'Ошибка при пополнении')
    }

    const bankWithdrawal = async (
        accountId: string,
        bankAccountId: string,
        amount: number,
        idempotencyKey: string
    ) => {
        return await handleAction(async () => {
            await accountsApi.bankWithdrawal({
                account_id: accountId,
                bank_account_id: bankAccountId,
                amount,
                idempotency_key: idempotencyKey
            })
            setSuccess('Запрос на вывод средств отправлен')
            await fetchAccounts()
        }, 'Ошибка при выводе средств')
    }

    const [statements, setStatements] = useState<Statement[]>([])

    const generateStatement = async (
        accountId: string,
        periodFrom: string,
        periodTo: string
    ) => {
        return await handleAction(async () => {
            const formattedFrom = `${periodFrom}T00:00:00Z`;
            const formattedTo = `${periodTo}T23:59:59Z`;
            const stmt = await accountsApi.generateStatement(accountId, {
                period_from: formattedFrom,
                period_to: formattedTo,
            })
            setStatements((prev) => [stmt, ...prev])
            setSuccess('Выписка сформирована и отправлена на email')
        }, 'Ошибка при формировании выписки')
    }

    const clearMessages = () => {
        setError(null)
        setSuccess(null)
    }

    return {
        accounts,
        activeAccounts,
        bankAccounts,
        loading,
        error,
        success,
        role,
        fetchAccounts,
        createAccount,
        deactivateAccount,
        linkBankAccount,
        bankDeposit,
        bankWithdrawal,
        statements,
        generateStatement,
        clearMessages,
    }
}

export const mapAccountsError = (e: unknown, defaultMsg: string): string => {
    const technical = mapTechnicalError(e);
    if (technical) return technical;

    if (e instanceof AxiosError && e.response) {
        const backendError = e.response.data?.error?.toLowerCase() || "";

        switch (e.response.status) {
            case 400:
                return "Неверный формат номера счета или БИК";
            case 404:
                if (backendError.includes("statement")) {
                    return "Операций за указанный период не найдено";
                }
                if (backendError.includes("bank account")) {
                    return "Банковский аккаунт не найден";
                }
                return "Ресурс не найден";
            default:
                return e.response.data?.error || defaultMsg;
        }
    }
    return defaultMsg;
};