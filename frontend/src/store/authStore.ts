import {create} from 'zustand'
import type {Role} from '@/models/auth'

interface AuthState {
    token: string | null
    userId: string | null
    companyId: string | null
    email: string | null
    role: Role | null
    isAuthenticated: boolean

    login: (data: {
        token: string
        user_id: string
        company_id: string
        email: string
        role: string
    }) => void
    logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
    token: localStorage.getItem('token'),
    userId: localStorage.getItem('userId'),
    companyId: localStorage.getItem('companyId'),
    email: localStorage.getItem('email'),
    role: localStorage.getItem('role') as Role | null,
    isAuthenticated: !!localStorage.getItem('token'),

    login: (data) => {
        localStorage.setItem('token', data.token)
        localStorage.setItem('userId', data.user_id)
        localStorage.setItem('companyId', data.company_id)
        localStorage.setItem('email', data.email)
        localStorage.setItem('role', data.role)
        set({
            token: data.token,
            userId: data.user_id,
            companyId: data.company_id,
            email: data.email,
            role: data.role as Role,
            isAuthenticated: true,
        })
    },

    logout: () => {
        localStorage.clear()
        set({
            token: null,
            userId: null,
            companyId: null,
            email: null,
            role: null,
            isAuthenticated: false,
        })
    },
}))