document.addEventListener('DOMContentLoaded', () => {
    // Проверяем, не авторизован ли уже
    if (localStorage.getItem('xsm_token')) {
        window.location.href = '/web/messenger.html';
        return;
    }

    // Переключение вкладок
    document.querySelectorAll('.auth-tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.auth-tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            tab.classList.add('active');
            const formId = tab.dataset.tab === 'login' ? 'login-form' : 'register-form';
            document.getElementById(formId).classList.add('active');
        });
    });

    // Переключение видимости пароля
    document.querySelectorAll('.toggle-pass').forEach(btn => {
        btn.addEventListener('click', () => {
            const input = btn.parentElement.querySelector('input');
            if (input) {
                const isPass = input.type === 'password';
                input.type = isPass ? 'text' : 'password';
                btn.innerHTML = isPass ? '<i class="fa-regular fa-eye-slash"></i>' : '<i class="fa-regular fa-eye"></i>';
            }
        });
    });

    // Индикатор надёжности пароля
    const regPass = document.getElementById('reg-password');
    if (regPass) {
        regPass.addEventListener('input', () => {
            const pass = regPass.value;
            let strength = 0;
            if (pass.length >= 8) strength++;
            if (/[A-Z]/.test(pass)) strength++;
            if (/[0-9]/.test(pass)) strength++;
            if (/[^A-Za-z0-9]/.test(pass)) strength++;
            
            const bar = document.querySelector('.strength-bar span');
            const colors = ['#ef4444', '#f59e0b', '#eab308', '#84cc16', '#22c55e'];
            const widths = ['0%', '25%', '50%', '75%', '100%'];
            if (bar) {
                bar.style.width = widths[strength];
                bar.style.background = colors[strength];
            }
        });
    }

    // Вход
    document.getElementById('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value.trim();
        const password = document.getElementById('login-password').value;
        const errorEl = document.getElementById('login-error');
        errorEl.textContent = '';

        if (!username || !password) {
            errorEl.textContent = 'Заполните все поля';
            return;
        }

        try {
            const res = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Ошибка входа');

            localStorage.setItem('xsm_token', data.access_token);
            window.location.href = '/web/messenger.html';
        } catch (err) {
            errorEl.textContent = err.message;
        }
    });

    // Регистрация
document.getElementById('register-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('reg-username').value.trim();
    const email = document.getElementById('reg-email').value.trim();
    const password = document.getElementById('reg-password').value;
    const errorEl = document.getElementById('register-error');
    errorEl.textContent = '';

    if (!username || !email || !password) {
        errorEl.textContent = 'Заполните все поля';
        return;
    }
    if (password.length < 8) {
        errorEl.textContent = 'Пароль должен быть не менее 8 символов';
        return;
    }

    try {
        const res = await fetch('/api/v1/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Ошибка регистрации');

        // Сохраняем токен и сразу переходим в мессенджер
        localStorage.setItem('xsm_token', data.access_token);
        
        // Показываем приветственное сообщение
        const submitBtn = document.querySelector('#register-form .btn-submit');
        const originalText = submitBtn.innerHTML;
        submitBtn.innerHTML = '<i class="fa-solid fa-check"></i> <span>Успешно!</span>';
        submitBtn.style.background = 'var(--success)';
        
        setTimeout(() => {
            window.location.href = '/web/messenger.html';
        }, 800);
        
    } catch (err) {
        errorEl.textContent = err.message;
    }
});
});