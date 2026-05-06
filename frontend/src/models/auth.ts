export interface AuthResponse {
    user_id: string
    company_id: string
    email: string
    role: string
    token: string
}

export interface UserResponse {
    user_id: string
    company_id: string
    login: string
    email: string
    role: string
}

export type Role = 'admin' | 'company_admin' | 'employee' | 'guest'