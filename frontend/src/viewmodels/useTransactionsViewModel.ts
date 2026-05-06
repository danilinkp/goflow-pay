import {useCallback, useState} from 'react'
import {transactionsApi} from '@/api/transactions'
import type {Transaction} from '@/models/transaction'
import {mapTechnicalError} from "@/utils/baseError.ts";
import {AxiosError} from "axios";

export function useTransactionsViewModel() {
    const [transactions, setTransactions] = useState<Transaction[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [success, setSuccess] = useState<string | null>(null)

    const handleAction = async (action: () => Promise<void>, defaultError: string) => {
        setLoading(true)
        setError(null)
        setSuccess(null)
        try {
            await action()
        } catch (e) {
            setError(mapTransactionsError(e, defaultError))
        } finally {
            setLoading(false)
        }
    }

    const fetchTransactions = useCallback(async (accountId: string) => {
        await handleAction(async () => {
            const txs = await transactionsApi.getTransactions(accountId)
            setTransactions(txs)
        }, 'Не удалось загрузить историю транзакций')
    }, [])

    const transfer = async (
        fromAccountId: string,
        toAccountId: string,
        amount: number,
        currency: string,
        idempotencyKey: string
    ) => {
        await handleAction(async () => {
            const tx = await transactionsApi.transfer({
                from_account_id: fromAccountId,
                to_account_id: toAccountId,
                amount,
                currency,
                idempotency_key: idempotencyKey,
            })
            setTransactions((prev) => [tx, ...prev])
            setSuccess(`Перевод выполнен успешно`)
        }, 'Ошибка при выполнении перевода')
    }

    const clearMessages = () => {
        setError(null)
        setSuccess(null)
    }

    return {
        transactions,
        loading,
        error,
        success,
        fetchTransactions,
        transfer,
        clearMessages,
    }
}

const mapTransactionsError = (e: unknown, defaultMsg: string): string => {
    const technical = mapTechnicalError(e);
    if (technical) return technical;

    if (e instanceof AxiosError && e.response) {
        switch (e.response.status) {
            case 404:
                return "Счет получателя не найден";
            case 422:
                return "Недостаточно средств для перевода";
            case 409:
                return "Транзакция уже была выполнена ранее";
            default:
                return e.response.data?.error || defaultMsg;
        }
    }
    return defaultMsg;
};