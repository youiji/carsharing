// Страница регистрации. После регистрации пользователь сразу входит в систему.
import { api, session } from '../api.js';
import { busy, flash, formErrors, formValues } from '../ui.js';

export function init() {
  const form = document.getElementById('register-form');
  const errors = formErrors(form);
  form.elements.full_name.focus();

  // Номер ВУ: только цифры, пробел после четвёртой.
  const license = form.elements.driver_license;
  license.addEventListener('input', () => {
    const digits = license.value.replace(/\D/g, '').slice(0, 10);
    license.value = digits.length > 4 ? `${digits.slice(0, 4)} ${digits.slice(4)}` : digits;
  });

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errors.clear();
    const v = formValues(form);

    // Быстрая проверка в браузере; окончательная — на сервере.
    const problems = {};
    if (v.full_name.length < 2) problems.full_name = 'Укажите ФИО';
    if (!v.email.includes('@')) problems.email = 'Укажите корректный email';
    if (v.driver_license.replace(/\D/g, '').length !== 10) problems.driver_license = 'Нужно 10 цифр';
    if (v.password.length < 8) problems.password = 'Не менее 8 символов';
    if (v.password !== v.password2) problems.password2 = 'Пароли не совпадают';
    if (Object.keys(problems).length) {
      errors.show({ message: 'Проверьте поля формы', details: problems });
      return;
    }

    const button = form.querySelector('[type="submit"]');
    busy(button, true);
    try {
      await api.post('/auth/register', {
        email: v.email, password: v.password, full_name: v.full_name, driver_license: v.driver_license,
      });
      const resp = await api.post('/auth/login', { email: v.email, password: v.password }, { redirectOn401: false });
      session.save(resp.token, resp.user);
      flash('Регистрация завершена. Добро пожаловать!', 'success');
      location.assign('/catalog');
    } catch (err) {
      errors.show(err);
    } finally {
      busy(button, false);
    }
  });
}
