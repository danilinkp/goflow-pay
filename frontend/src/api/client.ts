import axios from 'axios';

const BASE_URL = '/api/v1'

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
