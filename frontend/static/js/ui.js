// Общие элементы интерфейса: безопасное построение DOM, уведомления,
// ошибки форм, форматирование и подписи статусов.
//
// Защита от XSS: данные с сервера выводятся только через textContent
// (функция el и свойство text), innerHTML с данными не используется.

/**
 * Создаёт элемент. props: className, text, hidden, dataset, on (обработчики),
 * остальные ключи — атрибуты. Дети: узлы или строки (строки становятся текстом).
 */
export function el(tag, props = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(props)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === 'className') node.className = value;
    else if (key === 'text') node.textContent = value;
    else if (key === 'hidden') node.hidden = Boolean(value);
    else if (key === 'dataset') Object.assign(node.dataset, value);
    else if (key === 'on') for (const [ev, fn] of Object.entries(value)) node.addEventListener(ev, fn);
    else node.setAttribute(key, value === true ? '' : value);
  }
  for (const child of children.flat()) {
    if (child === undefined || child === null || child === false) continue;
    node.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
  return node;
}

// Заменяет содержимое узла.
export function render(node, ...children) {
  node.replaceChildren();
  for (const child of children.flat()) {
    if (child === undefined || child === null || child === false) continue;
    node.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
}

// ---------- Уведомления ----------

export function toast(message, type = 'info', timeout = 4500) {
  const box = document.querySelector('.toasts');
  if (!box) return;
  const item = el('div', { className: `toast toast-${type}`, role: type === 'error' ? 'alert' : 'status' },
    el('span', { text: message }),
    el('button', {
      className: 'toast-close', type: 'button', 'aria-label': 'Закрыть',
      on: { click: () => item.remove() },
    }, '×'));
  box.append(item);
  if (timeout) setTimeout(() => item.remove(), timeout);
}

// Сообщение, которое нужно показать после перехода на другую страницу.
export function flash(message, type = 'info') {
  try { sessionStorage.setItem('carsharing.flash', JSON.stringify({ message, type })); } catch { /* ignore */ }
}

export function showFlash() {
  try {
    const raw = sessionStorage.getItem('carsharing.flash');
    if (!raw) return;
    sessionStorage.removeItem('carsharing.flash');
    const { message, type } = JSON.parse(raw);
    toast(message, type);
  } catch { /* ignore */ }
}

// ---------- Формы ----------

// Значения полей формы (строки без пробелов по краям).
export function formValues(form) {
  const out = {};
  for (const [k, v] of new FormData(form).entries()) out[k] = typeof v === 'string' ? v.trim() : v;
  return out;
}

/**
 * Показ ошибок формы: details из ответа API — под соответствующими полями
 * ([data-error-for="поле"]), общий текст — в .form-error.
 */
export function formErrors(form) {
  const general = form.querySelector('.form-error');
  return {
    clear() {
      form.querySelectorAll('[data-error-for]').forEach((p) => { p.textContent = ''; });
      form.querySelectorAll('[aria-invalid]').forEach((i) => i.removeAttribute('aria-invalid'));
      if (general) general.textContent = '';
    },
    field(name, message) {
      const p = form.querySelector(`[data-error-for="${name}"]`);
      const input = form.elements[name];
      if (input) input.setAttribute('aria-invalid', 'true');
      const text = String(message);
      if (p) { p.textContent = text.charAt(0).toUpperCase() + text.slice(1); return true; }
      return false;
    },
    show(err) {
      const details = (err && err.details) || {};
      let shown = 0;
      for (const [name, message] of Object.entries(details)) {
        if (this.field(name, message)) shown++;
      }
      const msg = err && err.message ? err.message : 'Не удалось выполнить действие';
      if (!shown || Object.keys(details).length > shown) {
        if (general) general.textContent = msg; else toast(msg, 'error');
      }
      const first = form.querySelector('[aria-invalid="true"]');
      if (first) first.focus();
    },
  };
}

// Блокирует кнопку на время запроса.
export function busy(button, on) {
  if (!button) return;
  button.disabled = on;
  button.classList.toggle('is-busy', on);
}

// ---------- Форматирование ----------

const moneyFmt = new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 2 });
const dateTimeFmt = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' });
const dateFmt = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'long' });
const timeFmt = new Intl.DateTimeFormat('ru-RU', { timeStyle: 'short' });

export const formatMoney = (n) => (n === null || n === undefined ? '—' : moneyFmt.format(n));
export const formatDateTime = (iso) => (iso ? dateTimeFmt.format(new Date(iso)) : '—');
export const formatDate = (iso) => (iso ? dateFmt.format(new Date(iso)) : '—');
export const formatTime = (iso) => (iso ? timeFmt.format(new Date(iso)) : '—');

// ---------- Подписи ----------

export const ROLE_LABELS = { client: 'Клиент', manager: 'Менеджер', admin: 'Администратор' };

export const CAR_STATUS = {
  available: 'Свободен',
  booked: 'Забронирован',
  in_use: 'В поездке',
  maintenance: 'На обслуживании',
  decommissioned: 'Списан',
};

export const RENTAL_STATUS = {
  booked: 'Бронирование',
  active: 'В поездке',
  completed: 'Завершена',
  cancelled: 'Отменена',
};

// Цветная метка статуса: класс badge-<status>.
export function badge(status, labels) {
  return el('span', { className: `badge badge-${status}`, text: labels[status] || status });
}

// ---------- Модальные окна ----------

/**
 * Открывает модальное окно (<dialog>). content — узел или массив узлов.
 * Возвращает { dialog, body, close }. Окно удаляется из DOM после закрытия.
 */
export function openDialog({ title, content, className = '' }) {
  const body = el('div', { className: 'modal-body' }, content);
  const dialog = el('dialog', { className: `modal ${className}`, 'aria-labelledby': 'modal-title' },
    el('div', { className: 'modal-head' },
      el('h2', { id: 'modal-title', text: title }),
      el('button', {
        className: 'toast-close', type: 'button', 'aria-label': 'Закрыть',
        on: { click: () => dialog.close() },
      }, '×')),
    body);
  dialog.addEventListener('close', () => dialog.remove());
  document.body.append(dialog);
  dialog.showModal();
  return { dialog, body, close: () => dialog.close() };
}

/** Подтверждение действия. Возвращает Promise<boolean>. */
export function confirmDialog({ title, text, confirmText = 'Подтвердить', danger = false }) {
  return new Promise((resolve) => {
    let result = false;
    const ok = el('button', {
      className: `btn ${danger ? 'btn-danger' : 'btn-primary'}`, type: 'button',
      on: { click: () => { result = true; d.close(); } },
    }, confirmText);
    const d = openDialog({
      title,
      content: [
        ...[].concat(text).map((t) => (t instanceof Node ? t : el('p', { text: t }))),
        el('div', { className: 'form-actions' },
          el('button', { className: 'btn btn-secondary', type: 'button', on: { click: () => d.close() } }, 'Отмена'),
          ok),
      ],
    });
    d.dialog.addEventListener('close', () => resolve(result));
    ok.focus();
  });
}

// ---------- Таблицы ----------

/**
 * Сортировка по клику на заголовок: th[data-sort="поле"].
 * state — { sort, order }; onChange вызывается после смены.
 */
export function sortableHeaders(table, state, onChange) {
  const ths = [...table.querySelectorAll('th[data-sort]')];
  const paint = () => ths.forEach((th) => {
    th.classList.add('sortable');
    th.tabIndex = 0;
    if (th.dataset.sort === state.sort) th.setAttribute('aria-sort', state.order === 'desc' ? 'descending' : 'ascending');
    else th.removeAttribute('aria-sort');
  });
  const toggle = (th) => {
    if (state.sort === th.dataset.sort) state.order = state.order === 'asc' ? 'desc' : 'asc';
    else { state.sort = th.dataset.sort; state.order = 'asc'; }
    paint();
    onChange();
  };
  ths.forEach((th) => {
    th.addEventListener('click', () => toggle(th));
    th.addEventListener('keydown', (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggle(th); } });
  });
  paint();
  return paint;
}

// Строка «ничего не найдено» на всю ширину таблицы.
export function emptyRow(table, text) {
  const cols = table.querySelectorAll('thead th').length;
  return el('tr', {}, el('td', { className: 'table-empty', colspan: cols, text }));
}

export function debounce(fn, ms = 300) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

// «95 мин» → «1 ч 35 мин»
export function formatMinutes(min) {
  const h = Math.floor(min / 60);
  const m = min % 60;
  return h ? `${h} ч ${m} мин` : `${m} мин`;
}

// ---------- Формы в модальном окне ----------

let fieldSeq = 0;

/**
 * Форма в модальном окне по описанию полей:
 *   { name, label, type: 'text'|'number'|'select'|'textarea', options: [{value,label}],
 *     hint, wide, attrs: {min, max, step, maxlength, inputmode, placeholder...} }
 * onSubmit(values) — async; если бросает ApiError, ошибки показываются под полями,
 * иначе окно закрывается. extra — дополнительный узел под полями (например, карта).
 */
export function formDialog({ title, fields, values = {}, submitText = 'Сохранить', onSubmit, extra, className = '' }) {
  const inputs = {};
  const nodes = fields.map((f) => {
    const id = `f-${++fieldSeq}`;
    const attrs = { id, name: f.name, ...(f.attrs || {}) };
    let input;
    if (f.type === 'select') {
      input = el('select', attrs, f.options.map((o) =>
        el('option', { value: o.value, selected: String(values[f.name] ?? '') === String(o.value) }, o.label)));
    } else if (f.type === 'textarea') {
      input = el('textarea', { rows: 3, ...attrs });
      input.value = values[f.name] ?? '';
    } else {
      input = el('input', { type: f.type || 'text', ...attrs });
      input.value = values[f.name] ?? '';
    }
    inputs[f.name] = input;
    return el('div', { className: `field${f.wide ? ' field-wide' : ''}` },
      el('label', { for: id, text: f.label }),
      input,
      f.hint && el('p', { className: 'field-hint', text: f.hint }),
      el('p', { className: 'field-error', dataset: { errorFor: f.name } }));
  });

  const submit = el('button', { className: 'btn btn-primary', type: 'submit' }, submitText);
  const form = el('form', { className: 'form', novalidate: true },
    el('div', { className: 'form-grid' }, nodes),
    extra,
    el('p', { className: 'form-error', role: 'alert' }),
    el('div', { className: 'form-actions' },
      el('button', { className: 'btn btn-secondary', type: 'button', on: { click: () => d.close() } }, 'Отмена'),
      submit));
  const d = openDialog({ title, content: form, className });
  const errors = formErrors(form);
  (Object.values(inputs)[0])?.focus();

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errors.clear();
    const v = {};
    for (const [name, input] of Object.entries(inputs)) v[name] = input.value.trim();
    busy(submit, true);
    try {
      await onSubmit(v);
      d.close();
    } catch (err) {
      errors.show(err);
      busy(submit, false);
    }
  });
  return { ...d, form, inputs };
}

// Число из поля ввода: пусто → undefined (сервер сообщит «обязательное поле»), запятая → точка.
export function num(value) {
  if (value === undefined || value === null || String(value).trim() === '') return undefined;
  return Number(String(value).replace(',', '.'));
}

// Действие над строкой таблицы с подтверждением и обработкой ошибки.
export async function confirmAction({ title, text, confirmText, danger, action, success }) {
  if (!(await confirmDialog({ title, text, confirmText, danger }))) return false;
  try {
    await action();
    if (success) toast(success, 'success');
    return true;
  } catch (err) {
    toast(err.message, 'error', 8000);
    return false;
  }
}

// ---------- Фильтры в адресной строке ----------
// Обновление страницы, ссылка или кнопка «Назад» сохраняют фильтры и сортировку.

/** Заполняет state значениями из адресной строки (только известные ключи). */
export function readQuery(state) {
  const params = new URLSearchParams(location.search);
  for (const key of Object.keys(state)) {
    if (params.has(key)) state[key] = params.get(key);
  }
}

/** Записывает state в адресную строку, пропуская пустые значения и значения по умолчанию. */
export function writeQuery(state, defaults = {}) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(state)) {
    if (value !== '' && value !== null && value !== undefined && value !== defaults[key]) params.set(key, value);
  }
  const qs = params.toString();
  history.replaceState(null, '', `${location.pathname}${qs ? `?${qs}` : ''}`);
}
