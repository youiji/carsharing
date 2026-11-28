// Тарифы (менеджер, администратор): список, добавление, изменение, удаление.
import { api } from '../api.js';
import { confirmAction, el, emptyRow, formDialog, formatMoney, num, render, toast } from '../ui.js';

let table;

export async function init() {
  table = document.getElementById('tariffs-table');
  document.querySelector('[data-action="add"]').addEventListener('click', () => tariffDialog());
  await load();
}

async function load() {
  const tbody = table.querySelector('tbody');
  try {
    // Число автомобилей на тарифе — из каталога (фильтр tariff_id), одним запросом на весь автопарк.
    const [{ items }, cars] = await Promise.all([api.get('/tariffs'), api.get('/cars')]);
    const count = new Map();
    for (const c of cars.items) count.set(c.tariff.id, (count.get(c.tariff.id) || 0) + 1);
    render(tbody, items.length ? items.map((t) => row(t, count.get(t.id) || 0)) : emptyRow(table, 'Тарифов пока нет.'));
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

function row(t, cars) {
  return el('tr', {},
    el('td', { className: 'num', text: t.id }),
    el('td', {}, el('b', { text: t.name })),
    el('td', { className: 'num', text: formatMoney(t.price_per_min) }),
    el('td', { className: 'num', text: formatMoney(t.waiting_per_min) }),
    el('td', { className: 'num', text: String(cars) }),
    el('td', { className: 'actions' },
      el('button', { className: 'btn btn-secondary btn-sm', type: 'button', on: { click: () => tariffDialog(t) } }, 'Изменить'),
      el('button', { className: 'btn btn-ghost btn-sm', type: 'button', on: { click: () => remove(t) } }, 'Удалить')));
}

function tariffDialog(t) {
  formDialog({
    title: t ? `Тариф «${t.name}»` : 'Новый тариф',
    submitText: t ? 'Сохранить' : 'Добавить',
    values: t || {},
    fields: [
      { name: 'name', label: 'Название', wide: true, attrs: { maxlength: 50, required: true } },
      { name: 'price_per_min', label: 'Минута поездки, ₽', type: 'number', attrs: { min: 0, step: 0.01, inputmode: 'decimal' } },
      { name: 'waiting_per_min', label: 'Минута ожидания, ₽', type: 'number', attrs: { min: 0, step: 0.01, inputmode: 'decimal' } },
    ],
    onSubmit: async (v) => {
      const body = { name: v.name, price_per_min: num(v.price_per_min), waiting_per_min: num(v.waiting_per_min) };
      if (t) await api.put(`/tariffs/${t.id}`, body);
      else await api.post('/tariffs', body);
      toast(t ? 'Тариф изменён' : `Тариф «${v.name}» добавлен`, 'success');
      load();
    },
  });
}

async function remove(t) {
  if (await confirmAction({
    title: 'Удаление тарифа',
    text: [`Удалить тариф «${t.name}»?`, 'Тариф, который используют автомобили или аренды, удалить нельзя.'],
    confirmText: 'Удалить',
    danger: true,
    action: () => api.del(`/tariffs/${t.id}`),
    success: 'Тариф удалён',
  })) load();
}
