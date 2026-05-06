import {apiProvider} from './client'
import type { AuthResponse, UserResponse } from '@/models/auth'

export const authApi = {
    login: (email: string, password: string) =>
        apiProvider.post<AuthResponse>('/auth/login', { email, password }),

    registerCompany: (login: string, email: string, password: string, company_name: string) =>
        apiProvider.post<AuthResponse>('/auth/register/company', { login, email, password, company_name }),

    registerExisting: (login: string, email: string, password: string, invite_code: string) =>
        apiProvider.post<AuthResponse>('/auth/register', { login, email, password, invite_code }),

    logout: () =>
        apiProvider.post<void>('/auth/logout'),

    getUsers: (companyId?: string) => {
        const query = companyId ? `?company_id=${companyId}` : ''
        return apiProvider.get<UserResponse[]>(`/companies/users${query}`)
    },

    getInviteCode: (companyId?: string) => {
        const query = companyId ? `?company_id=${companyId}` : ''
        return apiProvider.get<{ invite_code: string }>(`/companies/invite-code${query}`)
    },
}