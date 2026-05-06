import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import LoginView from '@/views/LoginView'
import DashboardLayout from '@/views/DashboardLayout'
import AccountsView from '@/views/AccountsView'
import TransactionsView from '@/views/TransactionsView'
import CompanyView from '@/views/CompanyView'
import DashboardView from '@/views/DashboardView'

function PrivateRoute({ children }: { children: React.ReactNode }) {
    const { isAuthenticated } = useAuthStore()
    return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

export default function AppRouter() {
    return (
        <Routes>
            <Route path="/login" element={<LoginView />} />
            <Route
                path="/"
                element={
                    <PrivateRoute>
                        <DashboardLayout />
                    </PrivateRoute>
                }
            >
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<DashboardView />} />
                <Route path="accounts" element={<AccountsView />} />
                <Route path="transactions" element={<TransactionsView />} />
                <Route path="company" element={<CompanyView />} />
            </Route>
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
    )
}