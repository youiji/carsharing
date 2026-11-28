// Каркас страницы: сессия, меню по ролям, проверка доступа к странице,
// загрузка модуля страницы. Подключается на каждой странице из layout.html.
//
// Проверки ролей здесь — только для удобства (скрыть лишнее, перенаправить
// на вход). Данные защищает сервер: API проверяет токен и роль при каждом запросе.
import { api, session } from './api.js';
import { el, flash, render, showFlash, ROLE_LABELS } from './ui.js';

const body = document.body;

boot().catch((err) => {
  console.error(err);
  body.classList.remove('is-loading');
});

async function boot() {
  setupNavToggle();
  const user = await loadUser();
  applyVisibility(user);
  setupUserMenu(user);

  if (!checkAccess(user)) {
    body.classList.remove('is-loading');
    return;
  }
  body.classList.remove('is-loading');
  showFlash();

  const script = body.dataset.script;
  if (script) {
    const mod = await import(script);
    if (typeof mod.init === 'function') await mod.init({ user });
  }
}

// Пользователь берётся с сервера (GET /api/me): роль могла измениться
// с момента входа. Если токен истёк — продолжаем как гость.
async function loadUser() {
  if (!session.token()) return null;
  try {
    const user = await api.get('/me', { redirectOn401: false });
    session.setUser(user);
    return user;
  } catch (err) {
    if (err.status === 401) {
      session.clear();
      return null;
    }
    return session.user(); // сервер недоступен — показываем последнее известное состояние
  }
}

// Видимость по ролям:
//   data-roles (пункты меню): пусто — всем, "guest" — только гостю, иначе — перечисленным ролям;
//   data-show (элементы страниц): "guest", "user" (любой вошедший) или роли.
function applyVisibility(user) {
  const tokens = user ? ['user', user.role] : ['guest'];
  document.querySelectorAll('[data-roles]').forEach((node) => {
    if (node === body) return;
    const roles = node.dataset.roles.split(' ').filter(Boolean);
    node.hidden = roles.length > 0 && !roles.some((r) => tokens.includes(r));
  });
  document.querySelectorAll('[data-show]').forEach((node) => {
    node.hidden = !node.dataset.show.split(' ').some((r) => tokens.includes(r));
  });
}

function setupUserMenu(user) {
  const box = document.querySelector('.nav-user');
  if (!box || !user) return;
  box.hidden = false;
  box.querySelector('.nav-user-name').textContent = shortName(user.full_name);
  const role = box.querySelector('.nav-user-role');
  role.textContent = ROLE_LABELS[user.role] || user.role;
  role.classList.add(`badge-role-${user.role}`);
  box.querySelector('[data-action="logout"]').addEventListener('click', () => {
    session.clear();
    flash('Вы вышли из системы', 'info');
    location.assign('/');
  });
}

// «Петров Иван Сергеевич» → «Петров И. С.»
function shortName(full) {
  const [last, ...rest] = String(full || '').split(/\s+/).filter(Boolean);
  return [last, ...rest.map((p) => `${p[0]}.`)].join(' ');
}

// Доступ к странице по data-roles на <body>.
function checkAccess(user) {
  const roles = body.dataset.roles.split(' ').filter(Boolean);
  if (roles.length === 0) return true;

  if (roles.includes('guest')) {
    if (user) {
      location.replace('/');
      return false;
    }
    return true;
  }
  if (!user) {
    location.replace(`/login?next=${encodeURIComponent(location.pathname + location.search)}`);
    return false;
  }
  if (!roles.includes(user.role)) {
    document.querySelector('.page-content').hidden = true;
    const guard = document.querySelector('.page-guard');
    guard.hidden = false;
    render(guard, el('div', { className: 'error-page' },
      el('p', { className: 'error-code', text: '403' }),
      el('h1', { text: 'Недостаточно прав' }),
      el('p', { className: 'muted', text: 'Этот раздел недоступен для вашей роли.' }),
      el('a', { className: 'btn btn-primary', href: '/' }, 'На главную')));
    return false;
  }
  return true;
}

function setupNavToggle() {
  const btn = document.querySelector('.nav-toggle');
  const nav = document.getElementById('site-nav');
  if (!btn || !nav) return;
  btn.addEventListener('click', () => {
    const open = nav.classList.toggle('is-open');
    btn.setAttribute('aria-expanded', String(open));
  });
}
