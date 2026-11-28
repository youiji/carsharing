// Автомобили (менеджер, администратор): автопарк, добавление, изменение,
// смена статуса по таблице переходов (docs/api.md, 2.2) и удаление.
import { api, query } from '../api.js';
import {
  badge, CAR_STATUS, confirmAction, debounce, el, emptyRow, formDialog, formatMoney,
  num, openDialog, readQuery, render, sortableHeaders, toast, busy, writeQuery,
} from '../ui.js';
import { carIcon, createMap } from '../map.js';

// Ручные переходы статуса — копия таблицы 2.2 для подсказок в интерфейсе.
// Окончательно переход проверяет сервер.
const TRANSITIONS = {
  available: ['maintenance', 'decommissioned'],
  maintenance: ['available', 'decommissioned'],
  decommissioned: ['available'],
};

const DEFAULTS = { q: '', tariff_id: '', status: '', sort: 'id', order: 'asc' };
const state = { ...DEFAULTS };
let tariffs = [];
let table;

export async function init() {
  table = document.getElementById('fleet-table');
  const form = document.getElementById('filters');
  readQuery(state);
  form.q.value = state.q;
  form.status.value = state.status;
  form.addEventListener('submit', (e) => e.preventDefault());
  form.q.addEventListener('input', debounce(() => { state.q = form.q.value.trim(); load(); }));
  form.tariff_id.addEventListener('change', () => { state.tariff_id = form.tariff_id.value; load(); });
  form.status.addEventListener('change', () => { state.status = form.status.value; load(); });
  sortableHeaders(table, state, load);
  document.querySelector('[data-action="add"]').addEventListener('click', () => carDialog());

  try {
    tariffs = (await api.get('/tariffs')).items;
    for (const t of tariffs) form.tariff_id.append(el('option', { value: t.id }, t.name));
    form.tariff_id.value = state.tariff_id;
  } catch (err) { toast(err.message, 'error'); }
  await load();
}

async function load() {
  writeQuery(state, DEFAULTS);
  const tbody = table.querySelector('tbody');
  try {
    const { items, total } = await api.get(`/cars${query(state)}`);
    render(tbody, items.length ? items.map(row) : emptyRow(table, 'Автомобили не найдены.'));
    document.querySelector('.fleet-count').textContent = `Всего: ${total}`;
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

function row(car) {
  const locked = car.status === 'booked' || car.status === 'in_use';
  return el('tr', {},
    el('td', { className: 'num', text: car.id }),
    el('td', {}, el('b', { text: car.brand_model })),
    el('td', { className: 'nowrap', text: car.plate_number }),
    el('td', {}, car.tariff.name, el('span', { className: 'cell-sub', text: `${formatMoney(car.tariff.price_per_min)}/мин` })),
    el('td', { className: 'num', text: `${car.fuel_level}%` }),
    el('td', {}, badge(car.status, CAR_STATUS)),
    el('td', { className: 'actions' },
      el('button', { className: 'btn btn-secondary btn-sm', type: 'button', on: { click: () => carDialog(car) } }, 'Изменить'),
      el('button', {
        className: 'btn btn-secondary btn-sm', type: 'button', disabled: locked,
        title: locked ? 'Статус меняется только вместе с арендой' : null,
        on: { click: () => statusDialog(car) },
      }, 'Статус'),
      el('a', { className: 'btn btn-ghost btn-sm', href: `/maintenance?car_id=${car.id}` }, 'Журнал'),
      el('button', { className: 'btn btn-ghost btn-sm', type: 'button', on: { click: () => remove(car) } }, 'Удалить')));
}

// Добавление (car не передан) или изменение автомобиля. Место стоянки можно выбрать на карте.
function carDialog(car) {
  const mapBox = el('div', { className: 'map map-small' });
  const dlg = formDialog({
    title: car ? `Изменение: ${car.brand_model}, ${car.plate_number}` : 'Новый автомобиль',
    className: 'modal-wide',
    submitText: car ? 'Сохранить' : 'Добавить',
    values: car
      ? { ...car, tariff_id: car.tariff.id }
      : { fuel_level: 100, latitude: '55.75220', longitude: '37.61560', tariff_id: tariffs[0]?.id },
    fields: [
      { name: 'brand_model', label: 'Марка и модель', attrs: { maxlength: 100, required: true } },
      { name: 'plate_number', label: 'Гос. номер', hint: 'Пробелы и латиница допустимы: номер будет приведён к виду А101АА777', attrs: { maxlength: 20, required: true } },
      { name: 'tariff_id', label: 'Тариф', type: 'select', options: tariffs.map((t) => ({ value: t.id, label: `${t.name}, ${formatMoney(t.price_per_min)}/мин` })) },
      { name: 'fuel_level', label: 'Топливо, %', type: 'number', attrs: { min: 0, max: 100, step: 1, inputmode: 'numeric' } },
      { name: 'latitude', label: 'Широта', type: 'number', attrs: { min: -90, max: 90, step: 'any' } },
      { name: 'longitude', label: 'Долгота', type: 'number', attrs: { min: -180, max: 180, step: 'any' } },
    ],
    extra: el('div', { className: 'field' },
      el('span', { className: 'field-hint', text: 'Нажмите на карту, чтобы указать место стоянки.' }), mapBox),
    onSubmit: async (v) => {
      const body = {
        brand_model: v.brand_model, plate_number: v.plate_number, tariff_id: num(v.tariff_id),
        fuel_level: num(v.fuel_level), latitude: num(v.latitude), longitude: num(v.longitude),
      };
      const saved = car ? await api.put(`/cars/${car.id}`, body) : await api.post('/cars', body);
      toast(car ? 'Изменения сохранены' : `Автомобиль ${saved.plate_number} добавлен`, 'success');
      load();
    },
  });

  // Карта синхронизирована с полями координат.
  const { latitude, longitude } = dlg.inputs;
  const point = () => [num(latitude.value), num(longitude.value)];
  const map = createMap(mapBox, { center: point(), zoom: 14 });
  map.scrollWheelZoom.enable();
  const marker = window.L.marker(point(), { icon: carIcon(car ? car.status : 'available') }).addTo(map);
  map.on('click', (e) => {
    marker.setLatLng(e.latlng);
    latitude.value = e.latlng.lat.toFixed(6);
    longitude.value = e.latlng.lng.toFixed(6);
  });
  const sync = () => {
    const [lat, lng] = point();
    if (Number.isFinite(lat) && Number.isFinite(lng) && Math.abs(lat) <= 90 && Math.abs(lng) <= 180) {
      marker.setLatLng([lat, lng]);
      map.panTo([lat, lng]);
    }
  };
  latitude.addEventListener('change', sync);
  longitude.addEventListener('change', sync);
  setTimeout(() => map.invalidateSize(), 50);
  // Карта уничтожается вместе с окном: иначе каждое открытие оставляет в памяти
  // WebGL-контекст векторной подложки, и браузер начинает отключать старые.
  dlg.dialog.addEventListener('close', () => map.remove());
}

function statusDialog(car) {
  const targets = TRANSITIONS[car.status] || [];
  const options = targets.map((s, i) => el('label', { className: 'status-option' },
    el('input', { type: 'radio', name: 'status', value: s, checked: i === 0 }),
    badge(s, CAR_STATUS)));
  const submit = el('button', { className: 'btn btn-primary', type: 'submit' }, 'Изменить статус');
  const form = el('form', {},
    el('p', {}, 'Текущий статус: ', badge(car.status, CAR_STATUS)),
    el('div', { className: 'status-options', role: 'radiogroup', 'aria-label': 'Новый статус' }, options),
    el('p', { className: 'notice', text: 'Статусы «Забронирован» и «В поездке» устанавливаются только через аренду.' }),
    el('div', { className: 'form-actions' },
      el('button', { className: 'btn btn-secondary', type: 'button', on: { click: () => d.close() } }, 'Отмена'),
      submit));
  const d = openDialog({ title: `Статус: ${car.brand_model}, ${car.plate_number}`, content: form });
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const status = new FormData(form).get('status');
    busy(submit, true);
    try {
      await api.patch(`/cars/${car.id}/status`, { status });
      d.close();
      toast(`Статус изменён: ${CAR_STATUS[status]}`, 'success');
      load();
    } catch (err) {
      toast(err.message, 'error', 8000);
      busy(submit, false);
    }
  });
}

async function remove(car) {
  const done = await confirmAction({
    title: 'Удаление автомобиля',
    text: [`Удалить ${car.brand_model} (${car.plate_number})?`,
      'Автомобиль с историей аренд или обслуживания удалить нельзя: его можно перевести в статус «Списан».'],
    confirmText: 'Удалить',
    danger: true,
    action: () => api.del(`/cars/${car.id}`),
    success: 'Автомобиль удалён',
  });
  if (done) load();
}
