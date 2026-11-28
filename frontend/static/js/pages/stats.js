// Статистика (менеджер, администратор): ключевые показатели плитками
// и разбивки горизонтальными столбчатыми диаграммами Chart.js.
//
// Решения по визуализации:
//  - у каждой диаграммы одна величина → один цвет, без легенды (название — в заголовке);
//  - выручка и число аренд не совмещаются на одной диаграмме с двумя осями:
//    число аренд — в подсказке и в таблице данных;
//  - значение подписано у конца каждого столбца, у каждой диаграммы есть таблица данных
//    (доступность: значение не передаётся только длиной столбца).
import { api } from '../api.js';
import { CAR_STATUS, el, formatMoney, render, RENTAL_STATUS, toast } from '../ui.js';

// Цвета — hex-значения токенов из static/css/tokens.css (Chart.js разбирает цвета сам,
// поэтому OKLCH не передаётся). Цвет ряда проверен валидатором навыка dataviz на фоне --color-paper.
const SERIES = '#1f5ed9';        // --color-accent (кобальт)
const SERIES_HOVER = '#114cbf';  // --color-accent-hover
const TEXT = '#0d1f47';          // --color-ink
const MUTED = '#535e72';         // --color-muted
const GRID = '#d2d8e1';          // --color-rule
const ROLES = { client: 'Клиенты', manager: 'Менеджеры', admin: 'Администраторы' };
const charts = new Map();
const nf = new Intl.NumberFormat('ru-RU');

export async function init() {
  const Chart = window.Chart;
  Chart.defaults.font.family = getComputedStyle(document.body).fontFamily;
  Chart.defaults.font.size = 13;
  Chart.defaults.color = MUTED;
  document.querySelector('[data-action="refresh"]').addEventListener('click', load);
  await load();
}

async function load() {
  try {
    const s = await api.get('/stats');
    renderTiles(s);
    const count = (v) => nf.format(v);
    bar('revenue', 'Выручка по тарифам',
      s.revenue_by_tariff.map((t) => ({ label: t.tariff, value: t.revenue, extra: `${t.rentals_count} ${plural(t.rentals_count, 'аренда', 'аренды', 'аренд')}` })),
      (v) => formatMoney(v), ['Тариф', 'Аренд', 'Выручка']);
    bar('rentals', 'Аренды по статусам',
      Object.entries(RENTAL_STATUS).map(([k, label]) => ({ label, value: s.rentals_by_status[k] })), count, ['Статус', 'Аренд']);
    bar('cars', 'Автомобили по статусам',
      Object.entries(CAR_STATUS).map(([k, label]) => ({ label, value: s.cars_by_status[k] })), count, ['Статус', 'Автомобилей']);
    bar('users', 'Пользователи по ролям',
      Object.entries(ROLES).map(([k, label]) => ({ label, value: s.users_by_role[k] })), count, ['Роль', 'Пользователей']);
    document.querySelector('.stats-time').textContent =
      `Данные на ${new Date().toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}`;
  } catch (err) {
    toast(err.message, 'error');
  }
}

function renderTiles(s) {
  const carsTotal = Object.values(s.cars_by_status).reduce((a, b) => a + b, 0);
  const revenue = s.revenue_by_tariff.reduce((a, t) => a + Math.round(t.revenue * 100), 0) / 100;
  const completed = s.rentals_by_status.completed;
  render(document.querySelector('.stats-tiles'),
    tile('Пользователей', nf.format(s.users_total), `из них клиентов: ${s.users_by_role.client}`),
    tile('Свободно автомобилей', nf.format(s.cars_by_status.available), `всего в автопарке: ${carsTotal}`),
    tile('Сейчас в поездке', nf.format(s.rentals_by_status.active), `бронирований: ${s.rentals_by_status.booked}`),
    tile('Среднее время ожидания', s.avg_wait_minutes === null ? '—' : `${nf.format(s.avg_wait_minutes)} мин`, 'от бронирования до начала поездки'),
    tile('Выручка', formatMoney(revenue), `завершённых поездок: ${completed}, плюс платные отмены`));
}

function tile(label, value, hint) {
  return el('div', { className: 'tile' },
    el('span', { className: 'tile-label', text: label }),
    el('span', { className: 'tile-value', text: value }),
    el('span', { className: 'tile-hint', text: hint }));
}

/**
 * Горизонтальная столбчатая диаграмма + таблица данных.
 * rows: [{ label, value, extra? }], fmt — формат значения, head — заголовки таблицы.
 */
function bar(key, title, rows, fmt, head) {
  const box = document.querySelector(`[data-chart="${key}"]`);
  const summary = rows.map((r) => `${r.label}: ${fmt(r.value)}`).join(', ');
  const canvas = el('canvas', { role: 'img', 'aria-label': `${title}. ${summary}` });
  const wrap = el('div', { className: 'chart-canvas' }, canvas);
  wrap.style.height = `${rows.length * 40 + 36}px`; // element.style разрешён политикой CSP

  const table = el('details', { className: 'chart-data' },
    el('summary', { text: 'Таблица данных' }),
    el('table', {},
      el('thead', {}, el('tr', {}, head.map((h, i) => el('th', { className: i ? 'num' : '', scope: 'col', text: h })))),
      el('tbody', {}, rows.map((r) => el('tr', {},
        el('td', { text: r.label }),
        head.length === 3 && el('td', { className: 'num', text: r.extra }),
        el('td', { className: 'num', text: fmt(r.value) }))))));

  charts.get(key)?.destroy();
  render(box, wrap, table);

  const reduceMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
  charts.set(key, new window.Chart(canvas, {
    type: 'bar',
    data: {
      labels: rows.map((r) => r.label),
      datasets: [{
        data: rows.map((r) => r.value),
        backgroundColor: SERIES,
        hoverBackgroundColor: SERIES_HOVER,
        borderRadius: 0,          // тема Wayfind: прямые углы
        borderSkipped: 'start',
        maxBarThickness: 22,
        categoryPercentage: 0.85,
        barPercentage: 0.9,
      }],
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      animation: reduceMotion ? false : { duration: 400 },
      layout: { padding: { left: 6, right: 72 } },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: TEXT,
          padding: 10,
          displayColors: false,
          callbacks: {
            title: (items) => items[0].label,
            label: (ctx) => {
              const r = rows[ctx.dataIndex];
              return r.extra ? [fmt(r.value), r.extra] : fmt(r.value);
            },
          },
        },
      },
      scales: {
        x: {
          beginAtZero: true,
          grid: { color: GRID, drawTicks: false },
          border: { display: false },
          ticks: { precision: 0, padding: 6, maxTicksLimit: 4, maxRotation: 0, autoSkip: true, callback: (v) => (fmt === formatMoney ? `${nf.format(v)} ₽` : nf.format(v)) },
        },
        y: {
          grid: { display: false },
          border: { display: false },
          ticks: { color: TEXT, padding: 8 },
          // Chart.js ограничивает ширину оси половиной области, и на узком экране
          // длинные подписи («На обслуживании») обрезаются. Ширина задаётся по самой длинной подписи.
          afterFit(scale) {
            const { ctx } = scale;
            ctx.save();
            ctx.font = `${window.Chart.defaults.font.size}px ${window.Chart.defaults.font.family}`;
            const widest = Math.max(...rows.map((r) => ctx.measureText(r.label).width));
            ctx.restore();
            scale.width = Math.ceil(widest) + 16;
          },
        },
      },
    },
    plugins: [valueLabels(fmt)],
  }));
}

// Подпись значения у конца каждого столбца (цвет — текстовый, не цвет ряда).
function valueLabels(fmt) {
  return {
    id: 'valueLabels',
    afterDatasetsDraw(chart) {
      const { ctx } = chart;
      ctx.save();
      ctx.fillStyle = TEXT;
      ctx.font = `600 13px ${window.Chart.defaults.font.family}`;
      ctx.textBaseline = 'middle';
      chart.getDatasetMeta(0).data.forEach((bar, i) => {
        ctx.fillText(fmt(chart.data.datasets[0].data[i]), bar.x + 8, bar.y);
      });
      ctx.restore();
    },
  };
}

function plural(n, one, few, many) {
  const m10 = n % 10;
  const m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
}
