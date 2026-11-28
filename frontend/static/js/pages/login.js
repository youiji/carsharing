// Страница входа.
import { api, session } from '../api.js';
import { busy, flash, formErrors, formValues } from '../ui.js';

export function init() {
  const form = document.getElementById('login-form');
  const errors = formErrors(form);
  form.elements.email.focus();

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errors.clear();
    const { email, password } = formValues(form);
    if (!email) errors.field('email', 'Введите email');
    if (!password) errors.field('password', 'Введите пароль');
    if (!email || !password) return;

    const button = form.querySelector('[type="submit"]');
    busy(button, true);
    try {
      const resp = await api.post('/auth/login', { email, password }, { redirectOn401: false });
      session.save(resp.token, resp.user);
      flash(`Добро пожаловать, ${resp.user.full_name}!`, 'success');
      location.assign(nextPage());
    } catch (err) {
      errors.show(err);
    } finally {
      busy(button, false);
    }
  });
}

// Куда вернуться после входа. Принимается только адрес на этом же сайте.
// Адрес разбирается так же, как его разберёт браузер: проверка по началу строки
// обходится, например, «/<табуляция>/evil.example» — браузер удаляет табуляцию
// и получает «//evil.example», то есть чужой сайт.
function nextPage() {
  const next = new URLSearchParams(location.search).get('next');
  if (!next) return '/';
  try {
    const url = new URL(next, location.origin);
    if (url.origin === location.origin) return url.pathname + url.search + url.hash;
  } catch { /* некорректный адрес */ }
  return '/';
}
