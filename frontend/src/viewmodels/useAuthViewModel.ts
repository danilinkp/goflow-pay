import {useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {authApi} from '@/api/auth'
import {useAuthStore} from '@/store/authStore'
import {AxiosError} from "axios";
import {mapTechnicalError} from "@/utils/baseError.ts";

export function useAuthViewModel() {
    const navigate = useNavigate()
    const {login, logout, isAuthenticated, role, email, companyId} = useAuthStore()
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const handleAuthAction = async (
        action: () => Promise<any>,
        defaultMsg: string,
        context?: 'login' | 'register' | 'invite'
    ) => {
        setLoading(true)
        setError(null)
        try {
            const res = await action()
            login(res)
            navigate('/dashboard')
        } catch (e) {
            setError(mapAuthError(e, defaultMsg, context))
        } finally {
            setLoading(false)
        }
    }

    const handleLogin = (emailVal: string, password: string) =>
        handleAuthAction(
            () => authApi.login(emailVal, password),
            'Ошибка при входе',
            'login'
        )

    const handleRegisterCompany = (loginVal: string, emailVal: string, password: string, companyName: string) =>
        handleAuthAction(
            () => authApi.registerCompany(loginVal, emailVal, password, companyName),
            'Ошибка при регистрации компании',
            'register'
        )

    const handleRegisterExisting = (loginVal: string, emailVal: string, password: string, inviteCode: string) =>
        handleAuthAction(
            () => authApi.registerExisting(loginVal, emailVal, password, inviteCode),
            'Не удалось присоединиться к компании',
            'invite'
        )

    const handleLogout = async () => {
        try {
            await authApi.logout()
        } finally {
            logout()
            navigate('/login')
        }
    }

    return {
        isAuthenticated, role, email, companyId,
        loading, error,
        handleLogin, handleRegisterCompany, handleRegisterExisting, handleLogout,
    }
}

const mapAuthError = (e: unknown, defaultMsg: string, context?: 'login' | 'register' | 'invite'): string => {
    if (e instanceof AxiosError && e.response?.status === 401 && context === 'login') {
        return "Неверный логин или пароль";
    }

    const technical = mapTechnicalError(e);
    if (technical) return technical;

    if (e instanceof AxiosError && e.response) {
        const status = e.response.status;
        const serverMessage = e.response.data?.error;

        switch (status) {
            case 400:
                return "Проверьте корректность введенных данных";
            case 401:
                return "Неверный логин или пароль";
            case 404:
                return context === 'invite' ? "Код приглашения не найден" : "Пользователь не найден";
            case 409:
                return "Пользователь с таким Email или логином уже существует";
            default:
                return serverMessage && !serverMessage.includes("rpc error")
                    ? serverMessage
                    : defaultMsg;
        }
    }

    return defaultMsg;
};