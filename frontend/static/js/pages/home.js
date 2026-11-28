// Главная: приветствие и сводка по роли — текущая аренда клиента
// или ключевые показатели для менеджера и администратора.
import { api } from '../api.js';
import { badge, el, formatDateTime, formatMoney, formatTime, render, RENTAL_STATUS } from '../ui.js';

export async function init({ user }) {
  if (!user) return;
  document.querySelector('.hero-greeting').textContent = `Здравствуйте, ${user.full_name}!`;
  if (user.role === 'client') await currentRental();
  else await statsSummary();
}

async function currentRental() {
  const box = document.querySelector('.current-rental');
  try {
    const { items } = await api.get('/rentals/my?sort=id&order=desc');
    const open = items.find((r) => r.status === 'booked' || r.status === 'active');
    if (!open) {
      render(box, el('div', { className: 'empty-state' },
        el('p', { text: 'Сейчас у вас нет бронирований и поездок.' }),
        el('a', { className: 'btn btn-primary', href: '/catalog' }, 'Выбрать автомобиль')));
      return;
    }
    const since = open.status === 'active'
      ? `Поездка началась ${formatDateTime(open.started_at)}`
      : `Забронировано ${formatDateTime(open.booked_at)}, бронирование действует до ${formatTime(open.expires_at)}`;
    render(box, el('div', { className: 'card rental-card' },
      el('div', { className: 'rental-card-head' },
        el('h3', { text: `${open.car.brand_model}, ${open.car.plate_number}` }),
        badge(open.status, RENTAL_STATUS)),
      el('p', { className: 'muted', text: since }),
      el('p', { text: `Тариф: ${formatMoney(open.price_per_min)} / мин, ожидание ${formatMoney(open.waiting_per_min)} / мин` }),
      el('a', { className: 'btn btn-primary', href: '/rentals/my' }, 'Управлять арендой')));
  } catch (err) {
    render(box, el('p', { className: 'alert alert-error', text: err.message }));
  }
}

async function statsSummary() {
  const box = document.querySelector('.home-stats');
  try {
    const s = await api.get('/stats');
    // Сумма в копейках, как на странице статистики: без накопления ошибок дробных чисел.
    const revenue = s.revenue_by_tariff.reduce((sum, t) => sum + Math.round(t.revenue * 100), 0) / 100;
    render(box,
      tile('Свободно автомобилей', s.cars_by_status.available, `из ${Object.values(s.cars_by_status).reduce((a, b) => a + b, 0)}`),
      tile('Бронирований', s.rentals_by_status.booked),
      tile('В поездке', s.rentals_by_status.active),
      tile('На обслуживании', s.cars_by_status.maintenance),
      tile('Выручка', formatMoney(revenue), 'поездки и платные отмены'));
  } catch (err) {
    render(box, el('p', { className: 'alert alert-error', text: err.message }));
  }
}

function tile(label, value, hint) {
  return el('div', { className: 'tile' },
    el('span', { className: 'tile-label', text: label }),
    el('span', { className: 'tile-value', text: String(value) }),
    hint && el('span', { className: 'tile-hint', text: hint }));
}
