// Каталог: свободные автомобили (для клиента) или весь автопарк (для менеджера
// и администратора) в таблице с поиском и сортировкой и на карте. Клиент бронирует
// автомобиль из таблицы или из всплывающей подсказки на карте.
import { api, query } from '../api.js';
import {
  badge, CAR_STATUS, confirmDialog, debounce, el, emptyRow, flash, formatMoney,
  render, sortableHeaders, toast,
} from '../ui.js';
import { carClusterGroup, carIcon, createMap } from '../map.js';

let user;
let openRental = null; // текущая аренда клиента (бронирование или поездка)
let map;
let markers;
const markerById = new Map();
const state = { q: '', tariff_id: '', status: '', sort: 'id', order: 'asc' };
let firstFit = true;

export async function init(ctx) {
  user = ctx.user;
  const form = document.getElementById('filters');
  const table = document.getElementById('cars-table');

  readUrl(form);
  map = createMap(document.getElementById('map'));
  markers = carClusterGroup().addTo(map);

  const reload = () => { writeUrl(); load(table); };
  form.addEventListener('submit', (e) => e.preventDefault());
  form.q.addEventListener('input', debounce(() => { state.q = form.q.value.trim(); firstFit = true; reload(); }));
  form.tariff_id.addEventListener('change', () => { state.tariff_id = form.tariff_id.value; firstFit = true; reload(); });
  form.status.addEventListener('change', () => { state.status = form.status.value; firstFit = true; reload(); });
  form.addEventListener('reset', () => {
    setTimeout(() => {
      Object.assign(state, { q: '', tariff_id: '', status: '', sort: 'id', order: 'asc' });
      paintSort();
      firstFit = true;
      reload();
    });
  });
  const paintSort = sortableHeaders(table, state, reload);

  await Promise.all([loadTariffs(form), user.role === 'client' ? loadOpenRental() : null]);
  form.tariff_id.value = state.tariff_id;
  await load(table);
}

async function loadTariffs(form) {
  try {
    const { items } = await api.get('/tariffs');
    for (const t of items) {
      form.tariff_id.append(el('option', { value: t.id }, `${t.name}, ${formatMoney(t.price_per_min)}/мин`));
    }
  } catch (err) {
    toast(err.message, 'error');
  }
}

async function loadOpenRental() {
  try {
    const { items } = await api.get('/rentals/my?sort=id&order=desc');
    openRental = items.find((r) => r.status === 'booked' || r.status === 'active') || null;
  } catch { openRental = null; }
  const note = document.querySelector('.open-rental-note');
  if (openRental) {
    note.hidden = false;
    render(note,
      `У вас уже есть ${openRental.status === 'active' ? 'поездка' : 'бронирование'}: `,
      el('b', { text: `${openRental.car.brand_model}, ${openRental.car.plate_number}` }),
      '. Новое бронирование будет доступно после её завершения. ',
      el('a', { href: '/rentals/my' }, 'Перейти к аренде'));
  }
}

async function load(table) {
  const tbody = table.querySelector('tbody');
  const params = { q: state.q, tariff_id: state.tariff_id, sort: state.sort, order: state.order };
  if (user.role !== 'client') params.status = state.status;
  try {
    const { items, total } = await api.get(`/cars${query(params)}`);
    document.querySelector('.catalog-count').textContent = user.role === 'client'
      ? `Свободно автомобилей: ${total}`
      : `Автомобилей: ${total}`;
    render(tbody, items.length ? items.map(row) : emptyRow(table, 'Ничего не найдено. Измените условия поиска.'));
    drawMarkers(items);
  } catch (err) {
    render(tbody, emptyRow(table, err.message));
  }
}

function row(car) {
  const isClient = user.role === 'client';
  const tr = el('tr', { className: 'row-clickable', dataset: { id: car.id } },
    el('td', {}, el('b', { text: car.brand_model })),
    el('td', { className: 'nowrap', text: car.plate_number }),
    el('td', {}, fuel(car.fuel_level)),
    el('td', { text: car.tariff.name }),
    el('td', { className: 'num', text: formatMoney(car.tariff.price_per_min) }),
    el('td', { className: 'num', text: formatMoney(car.tariff.waiting_per_min) }),
    !isClient && el('td', {}, badge(car.status, CAR_STATUS)),
    isClient && el('td', { className: 'actions' }, bookButton(car, 'btn-sm')));
  tr.addEventListener('click', (e) => {
    if (e.target.closest('button')) return;
    focusCar(car.id);
  });
  return tr;
}

function fuel(level) {
  const cls = level < 20 ? 'fuel-low' : level < 50 ? 'fuel-mid' : 'fuel-ok';
  // Ширина задаётся через element.style: атрибут style в разметке запрещён политикой CSP.
  const fill = el('span', { className: 'fuel-fill' });
  fill.style.width = `${Math.max(0, Math.min(100, level))}%`;
  return el('span', { className: `fuel ${cls}`, title: `Топливо ${level}%` },
    el('span', { className: 'fuel-bar' }, fill),
    `${level}%`);
}

function bookButton(car, size = '') {
  const disabled = Boolean(openRental) || car.status !== 'available' || !user.driver_license;
  const title = openRental ? 'Сначала завершите текущую аренду'
    : !user.driver_license ? 'Для бронирования нужно водительское удостоверение' : null;
  return el('button', {
    className: `btn btn-primary ${size}`, type: 'button', disabled, title,
    on: { click: () => book(car) },
  }, 'Забронировать');
}

async function book(car) {
  const ok = await confirmDialog({
    title: 'Бронирование',
    text: [
      el('p', {}, el('b', { text: `${car.brand_model}, ${car.plate_number}` })),
      `Тариф «${car.tariff.name}»: ${formatMoney(car.tariff.price_per_min)} за минуту поездки.`,
      `Бронирование действует 20 минут. До начала поездки оплачивается ожидание: ${formatMoney(car.tariff.waiting_per_min)} за минуту,`
        + ' в том числе если вы отмените бронирование или оно истечёт.',
    ],
    confirmText: 'Забронировать',
  });
  if (!ok) return;
  try {
    await api.post('/rentals', { car_id: car.id });
    flash(`Автомобиль ${car.plate_number} забронирован`, 'success');
    location.assign('/rentals/my');
  } catch (err) {
    toast(err.message, 'error');
    if (err.code === 'car_unavailable') load(document.getElementById('cars-table'));
  }
}

// ---------- Карта ----------

function drawMarkers(cars) {
  markers.clearLayers();
  markerById.clear();
  for (const car of cars) {
    const m = window.L.marker([car.latitude, car.longitude], {
      icon: carIcon(car.status),
      title: `${car.brand_model}, ${car.plate_number}`,
      keyboard: true,
    }).bindPopup(() => popup(car), { minWidth: 220 });
    m.on('click', () => highlightRow(car.id));
    markers.addLayer(m);
    markerById.set(car.id, m);
  }
  if (firstFit && cars.length) {
    map.fitBounds(window.L.latLngBounds(cars.map((c) => [c.latitude, c.longitude])), { padding: [40, 40], maxZoom: 15 });
    firstFit = false;
  }
}

// Содержимое подсказки строится из DOM-узлов, а не из HTML-строки (защита от XSS).
function popup(car) {
  return el('div', { className: 'car-popup' },
    el('b', { text: car.brand_model }),
    el('div', { className: 'muted', text: car.plate_number }),
    el('div', { text: `Топливо: ${car.fuel_level}%` }),
    el('div', { text: `${car.tariff.name}: ${formatMoney(car.tariff.price_per_min)}/мин` }),
    user.role === 'client' ? bookButton(car, 'btn-sm') : badge(car.status, CAR_STATUS));
}

function focusCar(id) {
  const m = markerById.get(id);
  if (!m) return;
  highlightRow(id);
  // Если машина внутри группы, карта приближается до неё, затем открывается подсказка.
  markers.zoomToShowLayer(m, () => m.openPopup());
  document.getElementById('map').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function highlightRow(id) {
  document.querySelectorAll('#cars-table tr.is-selected').forEach((tr) => tr.classList.remove('is-selected'));
  document.querySelector(`#cars-table tr[data-id="${id}"]`)?.classList.add('is-selected');
}

// ---------- Параметры в адресной строке (обновление страницы сохраняет фильтры) ----------

function readUrl(form) {
  const p = new URLSearchParams(location.search);
  for (const k of Object.keys(state)) if (p.has(k)) state[k] = p.get(k);
  form.q.value = state.q;
  form.status.value = state.status;
}

function writeUrl() {
  const defaults = { sort: 'id', order: 'asc' };
  const params = Object.fromEntries(Object.entries(state).filter(([k, v]) => v && v !== defaults[k]));
  history.replaceState(null, '', `${location.pathname}${query(params)}`);
}
