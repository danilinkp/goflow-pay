import { AxiosError } from "axios";

export const mapTechnicalError = (e: unknown): string | null => {
    if (!(e instanceof AxiosError)) return null;
    if (!e.response) return "Сервер недоступен, проверьте соединение";

    switch (e.response.status) {
        case 401: return "Сессия истекла, войдите снова";
        case 500: return "Внутренняя ошибка сервера";
        default: return null;
    }
};