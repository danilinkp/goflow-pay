import {useState} from 'react'
import {useTransactionsViewModel} from '@/viewmodels/useTransactionsViewModel'
import {useAccountsViewModel} from '@/viewmodels/useAccountsViewModel'
import {useAuthStore} from '@/store/authStore'
import {v4 as uuidv4} from 'uuid'

const STATUS_MAP: Record<string, string> = {
    TRANSACTION_STATUS_PENDING: 'pending',
    TRANSACTION_STATUS_PROCESSING: 'pending',
    TRANSACTION_STATUS_SUCCESS: 'success',
    TRANSACTION_STATUS_FAILED: 'failed',
}

export default function TransactionsView() {
    const {role} = useAuthStore()
    const {transactions, loading, error, success, fetchTransactions, transfer, clearMessages} =
        useTransactionsViewModel()
    const {activeAccounts} = useAccountsViewModel()

    const [selectedAccountId, setSelectedAccountId] = useState('')
    const [modal, setModal] = useState(false)

    const [fromId, setFromId] = useState('')
    const [toId, setToId] = useState('')
    const [amount, setAmount] = useState('')
    const [currency, setCurrency] = useState('USD')

    const handleFetch = () => {
        if (selectedAccountId) fetchTransactions(selectedAccountId)
    }

    const handleTransfer = async () => {
        const success = await transfer(fromId, toId, Number(amount), currency, uuidv4())
        if (success) {
            setModal(false)
            if (fromId)
                fetchTransactions(fromId)
        }
    }

    const closeModal = () => {
        setModal(false)
        clearMessages()
    }

    const isCompanyAdmin = role === 'company_admin'

    return (
        <div>
            <div className="page-header">
                <div className="page-title">Transactions</div>
                <div className="page-subtitle">View and manage payment transactions</div>
            </div>

            {!modal && error && <div className="alert alert-error">{error}</div>}
            {success && <div className="alert alert-success">{success}</div>}

            <div className="card">
                <div className="card-title">Search by Account</div>
                <div style={{display: 'flex', gap: '8px', alignItems: 'flex-end'}}>
                    <div className="form-group" style={{flex: 1, marginBottom: 0}}>
                        <select
                            className="form-select"
                            value={selectedAccountId}
                            onChange={(e) => setSelectedAccountId(e.target.value)}
                        >
                            <option value="">Select account</option>
                            {activeAccounts.map((a) => (
                                <option key={a.account_id} value={a.account_id}>
                                    {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                </option>
                            ))}
                        </select>
                    </div>
                    <button className="btn btn-ghost" onClick={handleFetch} disabled={!selectedAccountId}>
                        Load
                    </button>
                    {isCompanyAdmin && (
                        <button className="btn btn-primary" onClick={() => setModal(true)}>
                            + Transfer
                        </button>
                    )}
                </div>
            </div>

            <div className="card">
                <div className="card-title">Transaction History</div>
                {loading ? (
                    <div className="loading">Loading...</div>
                ) : transactions.length === 0 ? (
                    <div className="empty-state">Select an account to view transactions</div>
                ) : (
                    <table className="table">
                        <thead>
                        <tr>
                            <th>ID</th>
                            <th>From</th>
                            <th>To</th>
                            <th>Amount</th>
                            <th>Status</th>
                            <th>Date</th>
                        </tr>
                        </thead>
                        <tbody>
                        {transactions.map((t) => (
                            <tr key={t.transaction_id}>
                                <td><span className="truncate"
                                          title={t.transaction_id}>{t.transaction_id.slice(0, 8)}...</span></td>
                                <td><span className="truncate"
                                          title={t.from_account_id}>{t.from_account_id.slice(0, 8)}...</span></td>
                                <td><span className="truncate"
                                          title={t.to_account_id}>{t.to_account_id.slice(0, 8)}...</span></td>
                                <td>{t.amount} {t.currency.replace('CURRENCY_', '')}</td>
                                <td>
            <span className={`badge badge-${STATUS_MAP[t.transaction_status] || 'pending'}`}>
                {t.transaction_status.replace('TRANSACTION_STATUS_', '')}
                </span>
                                </td>
                                <td>{new Date(t.created_at).toLocaleString()}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                )}
            </div>

            {modal && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">Transfer Funds</div>
                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        <div className="form-group">
                            <label className="form-label">From Account</label>
                            <select className="form-select" value={fromId} onChange={(e) => setFromId(e.target.value)}>
                                <option value="">Select source account</option>
                                {activeAccounts.map((a) => (
                                    <option key={a.account_id} value={a.account_id}>
                                        {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                        — {a.balance}
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="form-group">
                            <label className="form-label">To Account ID</label>
                            <input
                                className="form-input"
                                value={toId}
                                onChange={(e) => setToId(e.target.value)}
                                placeholder="Destination account UUID"
                            />
                        </div>
                        <div className="grid-2">
                            <div className="form-group">
                                <label className="form-label">Amount</label>
                                <input
                                    className="form-input"
                                    type="number"
                                    value={amount}
                                    onChange={(e) => setAmount(e.target.value)}
                                />
                            </div>
                            <div className="form-group">
                                <label className="form-label">Currency</label>
                                <select className="form-select" value={currency}
                                        onChange={(e) => setCurrency(e.target.value)}>
                                    {['EUR', 'USD', 'RUB', 'CNY'].map((c) => (
                                        <option key={c} value={c}>{c}</option>
                                    ))}
                                </select>
                            </div>
                        </div>
                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                            <button
                                className="btn btn-primary"
                                disabled={loading || !fromId || !toId || !amount}
                                onClick={handleTransfer}
                            >
                                {loading ? 'Processing...' : 'Transfer'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}