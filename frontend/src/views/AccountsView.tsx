import {useState} from 'react'
import {v4 as uuidv4} from 'uuid';
import {useAccountsViewModel} from '@/viewmodels/useAccountsViewModel'
import {useAuthStore} from '@/store/authStore'

const CURRENCIES = ['EUR', 'USD', 'RUB', 'CNY']

export default function AccountsView() {
    const {role} = useAuthStore()
    const {
        accounts, activeAccounts, bankAccounts, loading, error, success,
        fetchAccounts, createAccount, deactivateAccount,
        linkBankAccount, bankDeposit, bankWithdrawal, generateStatement,
        clearMessages,
    } = useAccountsViewModel()

    const [modal, setModal] = useState<string | null>(null)
    const [adminCompanyId, setAdminCompanyId] = useState('')

    const [currency, setCurrency] = useState('USD')
    const [lbAccountId, setLbAccountId] = useState('')
    const [lbName, setLbName] = useState('')
    const [lbBic, setLbBic] = useState('')
    const [lbSettlement, setLbSettlement] = useState('')
    const [lbCurrency, setLbCurrency] = useState('USD')
    const [opAccountId, setOpAccountId] = useState('')
    const [opBankAccountId, setOpBankAccountId] = useState('')
    const [opAmount, setOpAmount] = useState('')
    const [stmtAccountId, setStmtAccountId] = useState('')
    const [stmtFrom, setStmtFrom] = useState('')
    const [stmtTo, setStmtTo] = useState('')

    const isCompanyAdmin = role === 'company_admin'
    const isAdmin = role === 'admin'

    const closeModal = () => {
        setModal(null);
        clearMessages()
    }

    return (
        <div>
            <div className="page-header">
                <div className="page-title">Accounts</div>
                <div className="page-subtitle">Manage company accounts and bank connections</div>
            </div>

            {!modal && error && <div className="alert alert-error">{error}</div>}
            {!modal && success && <div className="alert alert-success">{success}</div>}

            {isAdmin && (
                <div className="card">
                    <div className="card-title">View by Company</div>
                    <div style={{display: 'flex', gap: '8px'}}>
                        <input
                            className="form-input"
                            style={{maxWidth: '360px'}}
                            value={adminCompanyId}
                            onChange={(e) => setAdminCompanyId(e.target.value)}
                            placeholder="Company ID (leave empty for own company)"
                        />
                        <button className="btn btn-ghost" onClick={() => fetchAccounts(adminCompanyId || undefined)}>
                            Load
                        </button>
                    </div>
                </div>
            )}

            <div className="actions-bar" style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: '12px',
                justifyContent: 'space-between',
                alignItems: 'center'
            }}>
                {isCompanyAdmin ? (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                        <button className="btn btn-primary" onClick={() => setModal('create')}>
                            + New Account
                        </button>
                        <button className="btn btn-ghost" onClick={() => setModal('link-bank')}>Link Bank</button>
                        <button className="btn btn-ghost" onClick={() => setModal('deposit')}>Deposit</button>
                        <button className="btn btn-ghost" onClick={() => setModal('withdrawal')}>Withdrawal</button>
                    </div>
                ) : (
                    <div />
                )}

                <div className="mobile-full-width" style={{ marginLeft: 'auto' }}>
                    <button
                        className="btn btn-outline"
                        onClick={() => setModal('statement')}
                        style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: '8px',
                            borderColor: '#007bff',
                            color: '#007bff'
                        }}
                    >
                        <span>📋</span> Generate statement
                    </button>
                </div>
            </div>

            <div className="card">
                <div className="card-title">Company Accounts</div>
                {loading ? <div className="loading">Loading...</div> : accounts.length === 0 ? (
                    <div className="empty-state">No accounts found</div>
                ) : (
                    <table className="table">
                        <thead>
                        <tr>
                            <th>Account ID</th>
                            <th>Currency</th>
                            <th>Balance</th>
                            <th>Status</th>
                            <th>Created</th>
                            {(isCompanyAdmin || isAdmin) && <th>Actions</th>}
                        </tr>
                        </thead>
                        <tbody>
                        {accounts.map((a) => (
                            <tr key={a.account_id}>
                                <td><span className="truncate" title={a.account_id}>{a.account_id}</span></td>
                                <td>{a.currency.replace('CURRENCY_', '')}</td>
                                <td>{a.balance}</td>
                                <td>
                    <span
                        className={`badge ${a.status.includes('ACTIVE') && !a.status.includes('IN') ? 'badge-active' : 'badge-inactive'}`}>
                      {a.status.replace('ACCOUNT_STATUS_', '')}
                    </span>
                                </td>
                                <td>{new Date(a.created_at).toLocaleDateString()}</td>
                                {(isCompanyAdmin || isAdmin) && (
                                    <td>
                                        {a.status.includes('ACTIVE') && !a.status.includes('IN') && (
                                            <button
                                                className="btn btn-danger btn-sm"
                                                onClick={() => deactivateAccount(
                                                    a.account_id,
                                                    isAdmin && adminCompanyId ? adminCompanyId : undefined
                                                )}
                                            >
                                                Deactivate
                                            </button>
                                        )}
                                    </td>
                                )}
                            </tr>
                        ))}
                        </tbody>
                    </table>
                )}
            </div>

            <div className="card">
                <div className="card-title">Bank Accounts</div>
                {bankAccounts.length === 0 ? <div className="empty-state">No bank accounts linked</div> : (
                    <table className="table">
                        <thead>
                        <tr>
                            <th>Name</th>
                            <th>BIC</th>
                            <th>Settlement Account</th>
                            <th>Currency</th>
                        </tr>
                        </thead>
                        <tbody>
                        {bankAccounts.map((ba) => (
                            <tr key={ba.bank_account_id}>
                                <td>{ba.name}</td>
                                <td>{ba.bic}</td>
                                <td><span className="truncate">{ba.settlement_account}</span></td>
                                <td>{ba.currency.replace('CURRENCY_', '')}</td>
                            </tr>
                        ))}
                        </tbody>
                    </table>
                )}
            </div>

            {modal === 'create' && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">New Account</div>
                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        <div className="form-group">
                            <label className="form-label">Currency</label>
                            <select className="form-select" value={currency}
                                    onChange={(e) => setCurrency(e.target.value)}>
                                {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
                            </select>
                        </div>
                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                            <button className="btn btn-primary" disabled={loading}
                                    onClick={async () => {
                                        const success = await createAccount(currency);
                                        if (success) {
                                            closeModal()
                                        }
                                    }}>
                                Create
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {modal === 'link-bank' && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">Link Bank Account</div>
                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        <div className="form-group">
                            <label className="form-label">Account</label>
                            <select className="form-select" value={lbAccountId}
                                    onChange={(e) => setLbAccountId(e.target.value)}>
                                <option value="">Select account</option>
                                {activeAccounts.map((a) => (
                                    <option key={a.account_id} value={a.account_id}>
                                        {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="grid-2">
                            <div className="form-group">
                                <label className="form-label">Name</label>
                                <input className="form-input" value={lbName}
                                       onChange={(e) => setLbName(e.target.value)}/>
                            </div>
                            <div className="form-group">
                                <label className="form-label">BIC</label>
                                <input className="form-input" value={lbBic} onChange={(e) => setLbBic(e.target.value)}/>
                            </div>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Settlement Account</label>
                            <input className="form-input" value={lbSettlement}
                                   onChange={(e) => setLbSettlement(e.target.value)}/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Currency</label>
                            <select className="form-select" value={lbCurrency}
                                    onChange={(e) => setLbCurrency(e.target.value)}>
                                {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
                            </select>
                        </div>
                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                            <button className="btn btn-primary" disabled={loading}
                                    onClick={async () => {
                                        const success = await linkBankAccount(lbAccountId, lbName, lbBic, lbSettlement, lbCurrency)
                                        if (success) {
                                            closeModal()
                                        }
                                    }}>
                                Link
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {modal === 'deposit' && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">Bank Deposit</div>
                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        <div className="form-group">
                            <label className="form-label">Account</label>
                            <select className="form-select" value={opAccountId}
                                    onChange={(e) => setOpAccountId(e.target.value)}>
                                <option value="">Select account</option>
                                {activeAccounts.map((a) => (
                                    <option key={a.account_id} value={a.account_id}>
                                        {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Bank Account</label>
                            <select className="form-select" value={opBankAccountId}
                                    onChange={(e) => setOpBankAccountId(e.target.value)}>
                                <option value="">Select bank account</option>
                                {bankAccounts.map((ba) => (
                                    <option key={ba.bank_account_id}
                                            value={ba.bank_account_id}>{ba.name} ({ba.bic})</option>
                                ))}
                            </select>
                        </div>
                        <div className="grid-2">
                            <div className="form-group">
                                <label className="form-label">Amount</label>
                                <input className="form-input" type="number" value={opAmount}
                                       onChange={(e) => setOpAmount(e.target.value)}/>
                            </div>
                        </div>
                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                            <button className="btn btn-primary" disabled={loading}
                                    onClick={async () => {
                                        const idempotencyKey = uuidv4();
                                        const success = await bankDeposit(opAccountId, opBankAccountId, Number(opAmount), idempotencyKey);
                                        if (success) {
                                            closeModal()
                                        }
                                    }}>
                                Deposit
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {modal === 'withdrawal' && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">Bank Withdrawal</div>
                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        <div className="form-group">
                            <label className="form-label">Account</label>
                            <select className="form-select" value={opAccountId}
                                    onChange={(e) => setOpAccountId(e.target.value)}>
                                <option value="">Select account</option>
                                {activeAccounts.map((a) => (
                                    <option key={a.account_id} value={a.account_id}>
                                        {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Bank Account</label>
                            <select className="form-select" value={opBankAccountId}
                                    onChange={(e) => setOpBankAccountId(e.target.value)}>
                                <option value="">Select bank account</option>
                                {bankAccounts.map((ba) => (
                                    <option key={ba.bank_account_id}
                                            value={ba.bank_account_id}>{ba.name} ({ba.bic})</option>
                                ))}
                            </select>
                        </div>
                        <div className="grid-2">
                            <div className="form-group">
                                <label className="form-label">Amount</label>
                                <input className="form-input" type="number" value={opAmount}
                                       onChange={(e) => setOpAmount(e.target.value)}/>
                            </div>
                        </div>
                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                            <button className="btn btn-primary" disabled={loading}
                                    onClick={async () => {
                                        const idempotencyKey = uuidv4();
                                        const success = await bankWithdrawal(opAccountId, opBankAccountId, Number(opAmount), idempotencyKey);
                                        if (success) {
                                            closeModal()
                                        }
                                    }}>
                                Withdraw
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {modal === 'statement' && (
                <div className="modal-overlay" onClick={closeModal}>
                    <div className="modal" onClick={(e) => e.stopPropagation()}>
                        <div className="modal-title">Выписка по счёту</div>

                        {error && <div className="alert alert-error mb-4">{error}</div>}
                        {success && <div className="alert alert-success mb-4">{success}</div>}

                        <div className="form-group">
                            <label className="form-label">Счёт</label>
                            <select
                                className="form-select"
                                value={stmtAccountId}
                                disabled={!!success}
                                onChange={(e) => setStmtAccountId(e.target.value)}
                            >
                                <option value="">Выберите счёт</option>
                                {accounts.map((a) => (
                                    <option key={a.account_id} value={a.account_id}>
                                        {a.account_id.slice(0, 8)}... ({a.currency.replace('CURRENCY_', '')})
                                    </option>
                                ))}
                            </select>
                        </div>
                        <div className="grid-2">
                            <div className="form-group">
                                <label className="form-label">С даты</label>
                                <input
                                    className="form-input"
                                    type="date"
                                    value={stmtFrom}
                                    disabled={!!success}
                                    onChange={(e) => setStmtFrom(e.target.value)}
                                />
                            </div>
                            <div className="form-group">
                                <label className="form-label">По дату</label>
                                <input
                                    className="form-input"
                                    type="date"
                                    value={stmtTo}
                                    disabled={!!success}
                                    onChange={(e) => setStmtTo(e.target.value)}
                                />
                            </div>
                        </div>

                        <div style={{ fontSize: '12px', color: 'var(--color-text-muted)', marginBottom: '16px' }}>
                            Выписка будет отправлена на ваш email
                        </div>

                        <div className="modal-footer">
                            <button className="btn btn-ghost" onClick={closeModal}>
                                {success ? 'Закрыть' : 'Отмена'}
                            </button>
                            <button
                                className={`btn ${success ? 'btn-success' : 'btn-primary'}`}
                                disabled={loading || !!success || !stmtAccountId || !stmtFrom || !stmtTo}
                                onClick={async () => {
                                    const ok = await generateStatement(stmtAccountId, stmtFrom, stmtTo);
                                    if (ok) {
                                        setTimeout(() => {
                                            closeModal();
                                        }, 2500);
                                    }
                                }}
                            >
                                {loading ? 'Формирование...' : success ? 'Отправлено ✔' : 'Сформировать и отправить'}
                            </button>
                        </div>
                    </div>
                </div>
            )}

        </div>
    )
}