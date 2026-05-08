import {useState} from 'react'
import {useAuthViewModel} from '@/viewmodels/useAuthViewModel'

type Tab = 'login' | 'register-company' | 'register'

export default function LoginView() {
    const [tab, setTab] = useState<Tab>('login')
    const {loading, error, handleLogin, handleRegisterCompany, handleRegisterExisting} =
        useAuthViewModel()

    // login form
    const [emailVal, setEmailVal] = useState('')
    const [password, setPassword] = useState('')

    // register company
    const [rcLogin, setRcLogin] = useState('')
    const [rcEmail, setRcEmail] = useState('')
    const [rcPassword, setRcPassword] = useState('')
    const [rcCompany, setRcCompany] = useState('')

    // register existing
    const [reLogin, setReLogin] = useState('')
    const [reEmail, setReEmail] = useState('')
    const [rePassword, setRePassword] = useState('')
    const [reCode, setReCode] = useState('')

    return (
        <div className="login-page">
            <div className="login-box">
                <div className="login-logo">GoFlow Pay</div>
                <div className="login-subtitle">B2B Payment System</div>

                <div className="login-tabs">
                    <button
                        className={`login-tab ${tab === 'login' ? 'active' : ''}`}
                        onClick={() => setTab('login')}
                    >
                        Login
                    </button>
                    <button
                        className={`login-tab ${tab === 'register-company' ? 'active' : ''}`}
                        onClick={() => setTab('register-company')}
                    >
                        New Company
                    </button>
                    <button
                        className={`login-tab ${tab === 'register' ? 'active' : ''}`}
                        onClick={() => setTab('register')}
                    >
                        Join Company
                    </button>
                </div>

                {error && <div className="alert alert-error">{error}</div>}

                {tab === 'login' && (
                    <form
                        onSubmit={(e) => {
                            e.preventDefault()
                            handleLogin(emailVal, password)
                        }}
                    >
                        <div className="form-group">
                            <label className="form-label">Email</label>
                            <input
                                className="form-input"
                                value={emailVal}
                                onChange={(e) => setEmailVal(e.target.value)}
                                placeholder="your_email"
                                required
                            />
                        </div>
                        <div className="form-group">
                            <label className="form-label">Password</label>
                            <input
                                className="form-input"
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                placeholder="••••••••"
                                required
                            />
                        </div>
                        <button className="btn btn-primary" style={{width: '100%'}} disabled={loading}>
                            {loading ? 'Signing in...' : 'Sign in'}
                        </button>
                    </form>
                )}

                {tab === 'register-company' && (
                    <form
                        onSubmit={(e) => {
                            e.preventDefault()
                            handleRegisterCompany(rcLogin, rcEmail, rcPassword, rcCompany)
                        }}
                    >
                        <div className="form-group">
                            <label className="form-label">Login</label>
                            <input className="form-input" value={rcLogin} onChange={(e) => setRcLogin(e.target.value)}
                                   required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Email</label>
                            <input className="form-input" type="email" value={rcEmail}
                                   onChange={(e) => setRcEmail(e.target.value)} required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Company Name</label>
                            <input className="form-input" value={rcCompany}
                                   onChange={(e) => setRcCompany(e.target.value)} required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Password</label>
                            <input className="form-input" type="password" value={rcPassword}
                                   onChange={(e) => setRcPassword(e.target.value)} required/>
                        </div>
                        <button className="btn btn-primary" style={{width: '100%'}} disabled={loading}>
                            {loading ? 'Creating...' : 'Create Company'}
                        </button>
                    </form>
                )}

                {tab === 'register' && (
                    <form
                        onSubmit={(e) => {
                            e.preventDefault()
                            handleRegisterExisting(reLogin, reEmail, rePassword, reCode)
                        }}
                    >
                        <div className="form-group">
                            <label className="form-label">Login</label>
                            <input className="form-input" value={reLogin} onChange={(e) => setReLogin(e.target.value)}
                                   required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Email</label>
                            <input className="form-input" type="email" value={reEmail}
                                   onChange={(e) => setReEmail(e.target.value)} required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Invite Code</label>
                            <input className="form-input" value={reCode} onChange={(e) => setReCode(e.target.value)}
                                   placeholder="XXXX-XXXX" required/>
                        </div>
                        <div className="form-group">
                            <label className="form-label">Password</label>
                            <input className="form-input" type="password" value={rePassword}
                                   onChange={(e) => setRePassword(e.target.value)} required/>
                        </div>
                        <button className="btn btn-primary" style={{width: '100%'}} disabled={loading}>
                            {loading ? 'Joining...' : 'Join Company'}
                        </button>
                    </form>
                )}
            </div>
        </div>
    )
}