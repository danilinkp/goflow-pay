import axios from 'axios';

const BASE_URL = '/api/v1'

// function getToken(): string | null {
//     return localStorage.getItem('token')
// }

export const api = axios.create({
    baseURL: BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
});

api.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token && config.headers) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

export const apiProvider = {
    get: <T>(path: string) => api.get<T>(path).then(res => res.data),
    post: <T>(path: string, body?: unknown) => api.post<T>(path, body).then(res => res.data),
    delete: <T>(path: string) => api.delete<T>(path).then(res => res.data),
};

// async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
//     const headers: HeadersInit = {
//         'Content-Type': 'application/json',
//     }
//
//     const token = getToken()
//     if (token) {
//         headers['Authorization'] = `Bearer ${token}`
//     }
//
//     const res = await fetch(`${BASE_URL}${path}`, {
//         method,
//         headers,
//         body: body ? JSON.stringify(body) : undefined,
//     })
//
//     if (!res.ok) {
//         const err = await res.json().catch(() => ({error: 'Unknown error'}))
//         throw new Error(err.error || `HTTP ${res.status}`)
//     }
//
//     if (res.status === 204 || res.headers.get('content-length') === '0') {
//         return undefined as T
//     }
//
//     return res.json()
// }
//
// export const api = {
//     get: <T>(path: string) => request<T>('GET', path),
//     post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
//     delete: <T>(path: string) => request<T>('DELETE', path),
// }