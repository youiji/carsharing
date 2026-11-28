// Пользователи и роли (администратор): поиск, фильтр по роли, смена роли,
// изменение ФИО, email и водительского удостоверения (аналог обращения
// в поддержку), удаление. Свою роль менять и себя удалять нельзя.
import { api, query } from '../api.js';
import {
  busy, confirmAction, debounce, el, emptyRow, formatDate, formDialog, openDialog,
  readQuery, render, ROLE_LABELS, sortableHeaders, toast, writeQuery,
} from '../ui.js';

const DEFAULTS = { q: '', role: '', sort: 'id', order: 'asc' };
const state = { ...DEFAULTS };
let me;
let table;

export async function init({ user }) {
  me = user;
  table = document.getElementById('users-table');
  const form = document.getElementById('filters');
  readQuery(state);
  form.q.value = state.q;
  form.role.value = state.role;
  form.addEventListener('submit', (e) => e.preventDefault());
  form.q.addEventListener('input', debounce(() => { state.q = form.q.value.trim(); load(); }));
  form.role.addEventListener('change', () => { state.role = form.role.value; load(); });
  sortableHeaders(table, state, load);
  await load();
}

async function load() {
  writeQuery(state, DEFAULTS);
  const tbody = table.querySelector('tbody');
  try {
    const { items, total } = await api.get(`/users${query(state)}`);
    render(tbody, items.length ? items.map(row) : emptyRow(table, 'Пользователи не найдены.'));
    document.querySelector('.users-count').textContent = `Найдено: ${total}`;
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

function row(u) {
  const self = u.id === me.id;
  return el('tr', {},
    el('td', { className: 'num', text: u.id }),
    el('td', {}, el('b', { text: u.full_name }), self && el('span', { className: 'cell-sub', text: 'это вы' })),
    el('td', { text: u.email }),
    el('td', { className: 'nowrap', text: u.driver_license || '—' }),
    el('td', {}, el('span', { className: `badge badge-role-${u.role}`, text: ROLE_LABELS[u.role] })),
    el('td', { className: 'nowrap', text: formatDate(u.created_at) }),
    el('td', { className: 'actions' },
      el('button', {
        className: 'btn btn-secondary btn-sm', type: 'button', disabled: self,
        title: self ? 'Свою роль изменить нельзя' : null, on: { click: () => roleDialog(u) },
      }, 'Роль'),
      el('button', { className: 'btn btn-secondary btn-sm', type: 'button', on: { click: () => dataDialog(u) } }, 'Данные'),
      // Аренды могут быть и у сотрудника, который раньше был клиентом.
      el('a', { className: 'btn btn-ghost btn-sm', href: `/rentals?user_id=${u.id}` }, 'Аренды'),
      el('button', {
        className: 'btn btn-ghost btn-sm', type: 'button', disabled: self,
        title: self ? 'Нельзя удалить свою учётную запись' : null, on: { click: () => remove(u) },
      }, 'Удалить')));
}

function roleDialog(u) {
  const options = Object.entries(ROLE_LABELS).map(([value, label]) => el('label', { className: 'status-option' },
    el('input', { type: 'radio', name: 'role', value, checked: value === u.role }),
    el('span', { className: `badge badge-role-${value}`, text: label }),
    el('span', { className: 'muted small', text: ROLE_HINTS[value] })));
  const submit = el('button', { className: 'btn btn-primary', type: 'submit' }, 'Сохранить');
  const form = el('form', {},
    el('div', { className: 'status-options', role: 'radiogroup', 'aria-label': 'Роль' }, options),
    el('p', { className: 'notice', text: 'Новая роль действует сразу, в том числе для уже выполненного входа. Пока у пользователя есть бронирование или поездка, роль изменить нельзя.' }),
    el('div', { className: 'form-actions' },
      el('button', { className: 'btn btn-secondary', type: 'button', on: { click: () => d.close() } }, 'Отмена'),
      submit));
  const d = openDialog({ title: `Роль: ${u.full_name}`, content: form });
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const role = new FormData(form).get('role');
    if (role === u.role) { d.close(); return; }
    busy(submit, true);
    try {
      await api.patch(`/users/${u.id}/role`, { role });
      d.close();
      toast(`${u.full_name}: роль «${ROLE_LABELS[role]}»`, 'success');
      load();
    } catch (err) {
      toast(err.message, 'error', 8000);
      busy(submit, false);
    }
  });
}

const ROLE_HINTS = {
  client: 'аренда автомобилей',
  manager: 'автопарк, тарифы, обслуживание, аренды, статистика',
  admin: 'права менеджера и управление пользователями',
};

// Изменение данных: отправляются только изменённые поля (PATCH).
function dataDialog(u) {
  formDialog({
    title: `Данные: ${u.full_name}`,
    values: { full_name: u.full_name, email: u.email, driver_license: u.driver_license || '' },
    fields: [
      { name: 'full_name', label: 'ФИО', wide: true, attrs: { maxlength: 100 } },
      { name: 'email', label: 'Email', type: 'email', attrs: { maxlength: 100 } },
      { name: 'driver_license', label: 'Водительское удостоверение', hint: '10 цифр', attrs: { maxlength: 11, inputmode: 'numeric' } },
    ],
    extra: el('p', { className: 'notice', text: 'Изменения вносятся после проверки документов вне системы. Пока у пользователя есть бронирование или поездка, данные изменить нельзя.' }),
    onSubmit: async (v) => {
      const body = {};
      if (v.full_name !== u.full_name) body.full_name = v.full_name;
      if (v.email !== u.email) body.email = v.email;
      if (v.driver_license !== (u.driver_license || '')) body.driver_license = v.driver_license;
      if (!Object.keys(body).length) { toast('Изменений нет', 'info'); return; }
      await api.patch(`/users/${u.id}`, body);
      toast('Данные пользователя изменены', 'success');
      load();
    },
  });
}

async function remove(u) {
  if (await confirmAction({
    title: 'Удаление пользователя',
    text: [`Удалить учётную запись ${u.full_name} (${u.email})?`,
      'Пользователя с арендами или записями обслуживания удалить нельзя: история должна сохраниться.'],
    confirmText: 'Удалить',
    danger: true,
    action: () => api.del(`/users/${u.id}`),
    success: 'Пользователь удалён',
  })) load();
}
