// Работа с REST API: токен, единый формат ошибок, реакция на истёкшую сессию.
import { flash } from './ui.js';

const TOKEN_KEY = 'carsharing.token';
const USER_KEY = 'carsharing.user';

// Хранилище сессии. Обращения к localStorage обёрнуты в try/catch:
// в приватном режиме браузера хранилище может быть недоступно.
export const session = {
  token() {
    try { return localStorage.getItem(TOKEN_KEY); } catch { return null; }
  },
  user() {
    try { return JSON.parse(localStorage.getItem(USER_KEY)); } catch { return null; }
  },
  save(token, user) {
    try {
      localStorage.setItem(TOKEN_KEY, token);
      localStorage.setItem(USER_KEY, JSON.stringify(user));
    } catch { /* сессия будет жить только до перезагрузки страницы */ }
  },
  setUser(user) {
    try { localStorage.setItem(USER_KEY, JSON.stringify(user)); } catch { /* ignore */ }
  },
  clear() {
    try {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(USER_KEY);
    } catch { /* ignore */ }
  },
};

// Ошибка API в формате docs/api.md, раздел 1.4.
export class ApiError extends Error {
  constructor(status, code, message, details = {}) {
    super(message);
    this.status = status;   // HTTP-код (0 — сервер недоступен)
    this.code = code;       // машинный код ошибки
    this.details = details; // поле → причина
  }
}

/**
 * Запрос к API. Возвращает разобранный JSON или null для 204.
 * При 401 с токеном сессия считается истёкшей: токен удаляется,
 * пользователь перенаправляется на вход (если не задано redirectOn401: false).
 */
export async function request(method, path, body, { redirectOn401 = true } = {}) {
  const token = session.token();
  const headers = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;

  let resp;
  try {
    resp = await fetch(`/api${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'network', 'Сервер недоступен. Проверьте подключение и повторите попытку.');
  }

  if (resp.status === 204) return null;
  let data = null;
  try { data = await resp.json(); } catch { /* тело не JSON */ }

  if (!resp.ok) {
    const e = (data && data.error) || {};
    const err = new ApiError(resp.status, e.code || `http_${resp.status}`,
      e.message || `Ошибка сервера (${resp.status})`, e.details || {});
    if (resp.status === 401 && token && redirectOn401) {
      session.clear();
      flash('Сессия истекла, войдите снова', 'error');
      location.assign(`/login?next=${encodeURIComponent(location.pathname + location.search)}`);
    }
    throw err;
  }
  return data;
}

export const api = {
  get: (path, opts) => request('GET', path, undefined, opts),
  post: (path, body, opts) => request('POST', path, body, opts),
  put: (path, body, opts) => request('PUT', path, body, opts),
  patch: (path, body, opts) => request('PATCH', path, body, opts),
  del: (path, opts) => request('DELETE', path, undefined, opts),
};

// Строка запроса из объекта; пустые значения пропускаются.
export function query(params) {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== '') q.set(k, v);
  }
  const s = q.toString();
  return s ? `?${s}` : '';
}
