import {useEffect, useState} from 'react'
import {authApi} from '@/api/auth'
import {useAuthStore} from '@/store/authStore'
import type {UserResponse} from '@/models/auth'

export default function CompanyView() {
    const {role, companyId, email} = useAuthStore()
    const [users, setUsers] = useState<UserResponse[]>([])
    const [inviteCode, setInviteCode] = useState<string | null>(null)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const [copied, setCopied] = useState(false)

    const [queryCompanyId, setQueryCompanyId] = useState('')

    const isAdmin = role === 'admin'
    const isCompanyAdmin = role === 'company_admin'
    const canManage = isAdmin || isCompanyAdmin

    const fetchUsers = async (cid?: string) => {
        setLoading(true)
        setError(null)
        try {
            const res = await authApi.getUsers(cid)
            setUsers(res)
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to fetch users')
        } finally {
            setLoading(false)
        }
    }

    const fetchInviteCode = async (cid?: string) => {
        setLoading(true)
        setError(null)
        try {
            const res = await authApi.getInviteCode(cid)
            setInviteCode(res.invite_code)
        } catch (e) {
            setError(e instanceof Error ? e.message : 'Failed to fetch invite code')
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        if (canManage) {
            fetchUsers()
            fetchInviteCode()
        }
    }, [])

    const handleCopy = () => {
        if (inviteCode) {
            navigator.clipboard.writeText(inviteCode)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
        }
    }

    return (
        <div>
            <div className="page-header">
                <div className="page-title">Company</div>
                <div className="page-subtitle">Manage users and company settings</div>
            </div>

            {error && <div className="alert alert-error">{error}</div>}

            {/* Invite Code */}
            {canManage && (
                <div className="card">
                    <div className="card-title">Invite Code</div>
                    {isAdmin && (
                        <div style={{display: 'flex', gap: '8px', marginBottom: '16px'}}>
                            <input
                                className="form-input"
                                style={{maxWidth: '320px'}}
                                value={queryCompanyId}
                                onChange={(e) => setQueryCompanyId(e.target.value)}
                                placeholder="Company ID (admin: query other company)"
                            />
                            <button
                                className="btn btn-ghost"
                                onClick={() => {
                                    fetchUsers(queryCompanyId || undefined)
                                    fetchInviteCode(queryCompanyId || undefined)
                                }}
                            >
                                Load
                            </button>
                        </div>
                    )}
                    {inviteCode ? (
                        <div style={{display: 'flex', gap: '8px', alignItems: 'center'}}>
                            <code style={{
                                background: 'var(--color-bg)',
                                border: '1px solid var(--color-border)',
                                borderRadius: 'var(--radius)',
                                padding: '8px 16px',
                                fontSize: '16px',
                                letterSpacing: '0.1em',
                                color: 'var(--color-primary)',
                                flex: 1,
                                maxWidth: '320px',
                            }}>
                                {inviteCode}
                            </code>
                            <button className="btn btn-ghost btn-sm" onClick={handleCopy}>
                                {copied ? 'Copied!' : 'Copy'}
                            </button>
                        </div>
                    ) : (
                        <div className="empty-state">No invite code available</div>
                    )}
                    <div style={{marginTop: '8px', fontSize: '12px', color: 'var(--color-text-muted)'}}>
                        Share this code with teammates to let them register in your company.
                    </div>
                </div>
            )}

            {/* Users */}
            {canManage && (
                <div className="card">
                    <div className="card-title">Team Members</div>
                    {loading ? (
                        <div className="loading">Loading...</div>
                    ) : users.length === 0 ? (
                        <div className="empty-state">No users found</div>
                    ) : (
                        <table className="table">
                            <thead>
                            <tr>
                                <th>Login</th>
                                <th>Email</th>
                                <th>Role</th>
                                <th>User ID</th>
                            </tr>
                            </thead>
                            <tbody>
                            {users.map((u) => (
                                <tr key={u.user_id}>
                                    <td>
                                        {u.email === email ? (
                                            <span>{u.login} <span style={{
                                                color: 'var(--color-text-muted)',
                                                fontSize: '11px'
                                            }}>(you)</span></span>
                                        ) : u.login}
                                    </td>
                                    <td>{u.email}</td>
                                    <td>
                      <span
                          className={`badge ${u.role === 'company_admin' || u.role === 'admin' ? 'badge-active' : 'badge-pending'}`}>
                        {u.role}
                      </span>
                                    </td>
                                    <td><span className="truncate" title={u.user_id}>{u.user_id.slice(0, 8)}...</span>
                                    </td>
                                </tr>
                            ))}
                            </tbody>
                        </table>
                    )}
                </div>
            )}

            {/* Info for employees */}
            {role === 'employee' && (
                <div className="card">
                    <div className="card-title">Your Account</div>
                    <div style={{color: 'var(--color-text-muted)', fontSize: '13px'}}>
                        <p>Email: {email}</p>
                        <p style={{marginTop: '8px'}}>Company ID: {companyId}</p>
                        <p style={{marginTop: '8px'}}>Role: {role}</p>
                    </div>
                </div>
            )}
        </div>
    )
}