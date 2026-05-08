import {useEffect} from 'react'
import {useNavigate} from 'react-router-dom'
import {useAuthStore} from '@/store/authStore'
import {useAccountsViewModel} from '@/viewmodels/useAccountsViewModel'

export default function DashboardView() {
    const {email, role} = useAuthStore()
    const {accounts, bankAccounts, loading, fetchAccounts} = useAccountsViewModel()
    const navigate = useNavigate()

    useEffect(() => {
        fetchAccounts()
    }, [])
    accounts.reduce((sum, a) => sum + a.balance, 0);
    const activeAccounts = accounts.filter((a) => a.status === 'ACCOUNT_STATUS_ACTIVE').length

    return (
        <div>
            <div className="page-header">
                <div className="page-title">Overview</div>
                <div className="page-subtitle">Welcome back, {email}</div>
            </div>

            <div className="stats-row">
                <div className="stat-card">
                    <div className="stat-label">Total Accounts</div>
                    <div className="stat-value">{accounts.length}</div>
                </div>
                <div className="stat-card">
                    <div className="stat-label">Active Accounts</div>
                    <div className="stat-value">{activeAccounts}</div>
                </div>
                <div className="stat-card">
                    <div className="stat-label">Bank Accounts</div>
                    <div className="stat-value">{bankAccounts.length}</div>
                </div>
                <div className="stat-card">
                    <div className="stat-label">Role</div>
                    <div className="stat-value" style={{fontSize: '16px'}}>{role}</div>
                </div>
            </div>

            {loading ? (
                <div className="loading">Loading...</div>
            ) : (
                <>
                    <div className="card">
                        <div className="card-title">Recent Accounts</div>
                        {accounts.length === 0 ? (
                            <div className="empty-state">No accounts yet</div>
                        ) : (
                            <table className="table">
                                <thead>
                                <tr>
                                    <th>Account ID</th>
                                    <th>Currency</th>
                                    <th>Balance</th>
                                    <th>Status</th>
                                </tr>
                                </thead>
                                <tbody>
                                {accounts.slice(0, 5).map((a) => (
                                    <tr key={a.account_id}>
                                        <td className="truncate">{a.account_id}</td>
                                        <td>{a.currency.replace('CURRENCY_', '')}</td>
                                        <td>{a.balance}</td>
                                        <td>
                        <span
                            className={`badge ${a.status.includes('ACTIVE') && !a.status.includes('IN') ? 'badge-active' : 'badge-inactive'}`}>
                          {a.status.replace('ACCOUNT_STATUS_', '')}
                        </span>
                                        </td>
                                    </tr>
                                ))}
                                </tbody>
                            </table>
                        )}
                    </div>

                    <div className="actions-bar">
                        <button className="btn btn-primary" onClick={() => navigate('/accounts')}>
                            Manage Accounts
                        </button>
                        <button className="btn btn-ghost" onClick={() => navigate('/transactions')}>
                            View Transactions
                        </button>
                    </div>
                </>
            )}
        </div>
    )
}