// Журнал обслуживания (менеджер, администратор). Менеджер изменяет и удаляет
// только свои записи, администратор — любые (окончательно проверяет сервер).
import { api, query } from '../api.js';
import {
  confirmAction, debounce, el, emptyRow, formatDateTime, formatMoney, formDialog, num,
  readQuery, render, sortableHeaders, toast, writeQuery,
} from '../ui.js';

const DEFAULTS = { q: '', car_id: '', sort: 'logged_at', order: 'desc' };
const state = { ...DEFAULTS };
let user;
let cars = [];
let table;

export async function init(ctx) {
  user = ctx.user;
  table = document.getElementById('maintenance-table');
  const form = document.getElementById('filters');
  readQuery(state);
  form.q.value = state.q;

  form.addEventListener('submit', (e) => e.preventDefault());
  form.q.addEventListener('input', debounce(() => { state.q = form.q.value.trim(); load(); }));
  form.car_id.addEventListener('change', () => { state.car_id = form.car_id.value; load(); });
  sortableHeaders(table, state, load);
  document.querySelector('[data-action="add"]').addEventListener('click', () => recordDialog());

  try {
    cars = (await api.get('/cars?sort=brand_model')).items;
    for (const c of cars) form.car_id.append(el('option', { value: c.id }, `${c.brand_model}, ${c.plate_number}`));
    form.car_id.value = state.car_id;
  } catch (err) { toast(err.message, 'error'); }
  await load();
}

async function load() {
  writeQuery(state, DEFAULTS);
  const tbody = table.querySelector('tbody');
  try {
    const { items, total } = await api.get(`/maintenance${query(state)}`);
    render(tbody, items.length ? items.map(row) : emptyRow(table, 'Записей не найдено.'));
    const sum = items.reduce((s, m) => s + Math.round(m.cost * 100), 0) / 100;
    document.querySelector('.maintenance-total').textContent = `Записей: ${total}, затраты: ${formatMoney(sum)}`;
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

const canEdit = (m) => user.role === 'admin' || m.manager.id === user.id;

function row(m) {
  return el('tr', {},
    el('td', { className: 'nowrap', text: formatDateTime(m.logged_at) }),
    el('td', {}, el('b', { text: m.car.brand_model }), el('span', { className: 'cell-sub', text: m.car.plate_number })),
    el('td', { className: 'desc-cell', text: m.description }),
    el('td', { className: 'num', text: formatMoney(m.cost) }),
    el('td', { text: m.manager.full_name }),
    el('td', { className: 'actions' }, canEdit(m)
      ? [
        el('button', { className: 'btn btn-secondary btn-sm', type: 'button', on: { click: () => recordDialog(m) } }, 'Изменить'),
        el('button', { className: 'btn btn-ghost btn-sm', type: 'button', on: { click: () => remove(m) } }, 'Удалить'),
      ]
      : el('span', { className: 'muted small', title: 'Менеджер изменяет только свои записи', text: 'чужая запись' })));
}

// Новая запись или изменение (автомобиль у записи не меняется).
function recordDialog(m) {
  const fields = [
    { name: 'description', label: 'Выполненные работы', type: 'textarea', wide: true, attrs: { maxlength: 255, required: true } },
    { name: 'cost', label: 'Стоимость, ₽', type: 'number', attrs: { min: 0, step: 0.01, inputmode: 'decimal' } },
  ];
  if (!m) {
    fields.unshift({
      name: 'car_id', label: 'Автомобиль', type: 'select', wide: true,
      options: [{ value: '', label: 'Выберите автомобиль' }, ...cars.map((c) => ({ value: c.id, label: `${c.brand_model}, ${c.plate_number}` }))],
    });
  }
  formDialog({
    title: m ? `Запись: ${m.car.brand_model}, ${m.car.plate_number}` : 'Новая запись обслуживания',
    submitText: m ? 'Сохранить' : 'Добавить',
    values: m || { car_id: state.car_id },
    fields,
    onSubmit: async (v) => {
      if (m) {
        await api.patch(`/maintenance/${m.id}`, { description: v.description, cost: num(v.cost) });
        toast('Запись изменена', 'success');
      } else {
        await api.post('/maintenance', { car_id: num(v.car_id) ?? 0, description: v.description, cost: num(v.cost) });
        toast('Запись добавлена. Статус автомобиля при необходимости измените в разделе «Автомобили».', 'success', 7000);
      }
      load();
    },
  });
}

async function remove(m) {
  if (await confirmAction({
    title: 'Удаление записи',
    text: `Удалить запись «${m.description}» (${m.car.plate_number})?`,
    confirmText: 'Удалить',
    danger: true,
    action: () => api.del(`/maintenance/${m.id}`),
    success: 'Запись удалена',
  })) load();
}
