// Все аренды (менеджер, администратор): поиск по ФИО и гос. номеру, фильтр
// по статусу, сортировка, отмена бронирований клиентов.
import { api, query } from '../api.js';
import {
  badge, confirmAction, debounce, el, emptyRow, formatDateTime, formatMoney, render,
  readQuery, RENTAL_STATUS, sortableHeaders, writeQuery,
} from '../ui.js';

const DEFAULTS = { q: '', status: '', user_id: '', car_id: '', sort: 'id', order: 'desc' };
const state = { ...DEFAULTS };
let table;

export async function init() {
  table = document.getElementById('rentals-table');
  const form = document.getElementById('filters');
  readQuery(state);
  form.q.value = state.q;
  form.status.value = state.status;

  form.addEventListener('submit', (e) => e.preventDefault());
  form.q.addEventListener('input', debounce(() => { state.q = form.q.value.trim(); load(); }));
  form.status.addEventListener('change', () => { state.status = form.status.value; load(); });
  sortableHeaders(table, state, load);
  await load();
}

async function load() {
  writeQuery(state, DEFAULTS);
  const tbody = table.querySelector('tbody');
  try {
    const { items, total } = await api.get(`/rentals${query(state)}`);
    render(tbody, items.length ? items.map(row) : emptyRow(table, 'Аренды не найдены.'));
    showFilterNote(items);
    // Начислено: стоимость завершённых поездок и списанное при отмене ожидание.
    const charged = items.reduce((s, r) => s + (r.total_cost ? Math.round(r.total_cost * 100) : 0), 0) / 100;
    document.querySelector('.rentals-summary').textContent = `Найдено: ${total}, начислено: ${formatMoney(charged)}`;
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

// Пояснение при переходе по ссылке «Аренды пользователя» или «Аренды автомобиля»:
// имя клиента или автомобиль берутся из найденных аренд.
function showFilterNote(items) {
  const note = document.querySelector('.filter-note');
  if (!state.user_id && !state.car_id) {
    note.hidden = true;
    return;
  }
  let subject;
  if (state.user_id) {
    subject = items[0] ? `клиента ${items[0].user.full_name}` : `пользователя №${state.user_id}`;
  } else {
    subject = items[0] ? `автомобиля ${items[0].car.brand_model}, ${items[0].car.plate_number}` : `автомобиля №${state.car_id}`;
  }
  note.hidden = false;
  render(note, `Показаны аренды ${subject}. `,
    el('a', { href: '/rentals', on: { click: (e) => { e.preventDefault(); state.user_id = ''; state.car_id = ''; load(); } } }, 'Показать все'));
}

function row(r) {
  return el('tr', {},
    el('td', { className: 'num', text: r.id }),
    el('td', {}, el('a', { href: `/rentals?user_id=${r.user.id}`, text: r.user.full_name })),
    el('td', {}, el('b', { text: r.car.brand_model }), el('span', { className: 'cell-sub', text: r.car.plate_number })),
    el('td', { className: 'nowrap', text: formatDateTime(r.booked_at) }),
    el('td', { className: 'nowrap', text: r.started_at ? `${formatDateTime(r.started_at)} – ${r.ended_at ? formatDateTime(r.ended_at) : '…'}` : '—' }),
    el('td', { className: 'num', text: r.total_cost === null ? '—' : formatMoney(r.total_cost) }),
    el('td', {}, badge(r.status, RENTAL_STATUS)),
    el('td', { className: 'actions' }, r.status === 'booked' && el('button', {
      className: 'btn btn-ghost btn-sm', type: 'button', on: { click: () => cancel(r) },
    }, 'Отменить')));
}

async function cancel(r) {
  if (await confirmAction({
    title: 'Отмена бронирования',
    text: [`Отменить бронирование клиента ${r.user.full_name}: ${r.car.brand_model} (${r.car.plate_number})?`,
      'Отмена сотрудником бесплатна для клиента: ожидание не списывается. Автомобиль станет свободен.'],
    confirmText: 'Отменить бронирование',
    danger: true,
    action: () => api.post(`/rentals/${r.id}/cancel`),
    success: 'Бронирование отменено',
  })) load();
}
