import {Outlet, useLocation, useNavigate} from 'react-router-dom'
import {useAuthStore} from '@/store/authStore'
import {authApi} from '@/api/auth'

const navItems = [
    {path: '/dashboard', label: 'Overview'},
    {path: '/accounts', label: 'Accounts'},
    {path: '/transactions', label: 'Transactions'},
    {path: '/company', label: 'Company'},
]

export default function DashboardLayout() {
    const navigate = useNavigate()
    const location = useLocation()
    const {email, role, logout} = useAuthStore()

    const handleLogout = async () => {
        try {
            await authApi.logout()
        } finally {
            logout()
            navigate('/login')
        }
    }

    return (
        <div className="layout">
            <aside className="sidebar">
                <div className="sidebar-logo">GoFlow Pay</div>
                <nav className="sidebar-nav">
                    {navItems.map((item) => (
                        <button
                            key={item.path}
                            className={`nav-item ${location.pathname === item.path ? 'active' : ''}`}
                            onClick={() => navigate(item.path)}
                        >
                            {item.label}
                        </button>
                    ))}
                </nav>
                <div className="sidebar-footer">
                    <div className="user-info">
                        <div>{email}</div>
                        <span className="user-role">{role}</span>
                    </div>
                    <button className="btn btn-ghost btn-sm" style={{width: '100%'}} onClick={handleLogout}>
                        Sign out
                    </button>
                </div>
            </aside>
            <main className="main-content">
                <Outlet/>
            </main>
        </div>
    )
}