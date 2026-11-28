// Профиль: личные данные (только чтение) и смена пароля.
import { api } from '../api.js';
import { busy, el, formatDate, formErrors, formValues, render, ROLE_LABELS, toast } from '../ui.js';

export async function init({ user }) {
  render(document.querySelector('.profile-data'),
    prop('ФИО', user.full_name),
    prop('Email', user.email),
    prop('Водительское удостоверение', user.driver_license || 'не указано'),
    prop('Роль', ROLE_LABELS[user.role] || user.role),
    prop('Дата регистрации', formatDate(user.created_at)));

  const form = document.getElementById('password-form');
  const errors = formErrors(form);
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errors.clear();
    const v = formValues(form);
    if (!v.current_password) { errors.field('current_password', 'Введите текущий пароль'); return; }
    if (v.new_password.length < 8) { errors.field('new_password', 'Не менее 8 символов'); return; }
    if (v.new_password !== v.new_password2) { errors.field('new_password2', 'Пароли не совпадают'); return; }

    const button = form.querySelector('[type="submit"]');
    busy(button, true);
    try {
      await api.put('/me/password', { current_password: v.current_password, new_password: v.new_password });
      form.reset();
      toast('Пароль изменён', 'success');
    } catch (err) {
      if (err.code === 'wrong_password') errors.field('current_password', err.message);
      else errors.show(err);
    } finally {
      busy(button, false);
    }
  });
}

function prop(name, value) {
  return [el('dt', { text: name }), el('dd', { text: value })];
}
