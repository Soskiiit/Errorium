/**
 * Errorium Web Admin Panel (FR-5)
 * Zero-dependency vanilla JS admin UI mounted at /admin.
 * Guarded behind isAdmin() — non-admins and visitors never execute heavy work.
 */

(function () {
    'use strict';

    var LANG_KEY = 'errorium_app_language';
    var VIEW_ID = 'admin-view';
    var ROOT_ID = 'admin-root';
    var SIDEBAR_LINKS = 'div.sidebar-links-left';

    var L = {
        'Русский': {
            admin: 'Админ',
            title: 'Панель администратора',
            subtitle: 'Текущий пользователь',
            tabDashboard: 'Дашборд',
            tabUsers: 'Пользователи',
            tabRag: 'RAG-конфиг',
            tabIndex: 'Индекс',
            dashboard: 'Дашборд',
            documents: 'Документы',
            vocabulary: 'Словарь',
            avgDocLen: 'Средняя длина док.',
            indexSize: 'Размер индекса',
            coreReady: 'Ядро готово',
            yes: 'да',
            no: 'нет',
            latency: 'Задержка',
            queriesTotal: 'Запросов всего',
            loading: 'Загрузка…',
            loadError: 'Ошибка загрузки статистики',
            users: 'Пользователи',
            searchPlaceholder: 'Поиск (username / email)…',
            load: 'Загрузить',
            username: 'Username',
            email: 'Email',
            role: 'Роль',
            active: 'Активен',
            lastLogin: 'Последний вход',
            actions: 'Действия',
            makeAdmin: 'Сделать админом',
            makeUser: 'Сделать пользователем',
            block: 'Заблокировать',
            unblock: 'Разблокировать',
            confirmRole: 'Сменить роль пользователю {u} на {r}?',
            roleChanged: 'Роль обновлена',
            roleChangeFailed: 'Ошибка смены роли',
            blocked: 'Пользователь заблокирован',
            unblocked: 'Пользователь разблокирован',
            blockFailed: 'Ошибка блокировки',
            usersLoadFailed: 'Ошибка загрузки пользователей',
            rag: 'RAG-конфиг',
            provider: 'Провайдер',
            baseUrl: 'Base URL',
            model: 'Модель',
            systemPrompt: 'Системный промпт',
            temperature: 'Temperature',
            maxTokens: 'Max tokens',
            save: 'Сохранить',
            ragSaved: 'RAG-конфиг сохранён',
            ragSaveFailed: 'Ошибка сохранения RAG-конфига',
            ragLoadFailed: 'Ошибка загрузки RAG-конфига',
            index: 'Индекс',
            reloadIndex: 'Перезагрузить индекс',
            reloadOk: 'Индекс перезагружен: {ts}',
            reloadFailed: 'Ошибка перезагрузки индекса',
            coreUnavailable: 'Поисковое ядро недоступно (502)',
            toastSaved: 'Сохранено',
            accessOnly: 'Только для администратора'
        },
        'English': {
            admin: 'Admin',
            title: 'Admin panel',
            subtitle: 'Current user',
            tabDashboard: 'Dashboard',
            tabUsers: 'Users',
            tabRag: 'RAG config',
            tabIndex: 'Index reload',
            dashboard: 'Dashboard',
            documents: 'Documents',
            vocabulary: 'Vocabulary',
            avgDocLen: 'Avg doc len',
            indexSize: 'Index size',
            coreReady: 'Core ready',
            yes: 'yes',
            no: 'no',
            latency: 'Latency',
            queriesTotal: 'Queries total',
            loading: 'Loading…',
            loadError: 'Failed to load stats',
            users: 'Users',
            searchPlaceholder: 'Search (username / email)…',
            load: 'Load',
            username: 'Username',
            email: 'Email',
            role: 'Role',
            active: 'Active',
            lastLogin: 'Last login',
            actions: 'Actions',
            makeAdmin: 'Make admin',
            makeUser: 'Make user',
            block: 'Block',
            unblock: 'Unblock',
            confirmRole: 'Change role for {u} to {r}?',
            roleChanged: 'Role updated',
            roleChangeFailed: 'Failed to change role',
            blocked: 'User blocked',
            unblocked: 'User unblocked',
            blockFailed: 'Failed to block user',
            usersLoadFailed: 'Failed to load users',
            rag: 'RAG config',
            provider: 'Provider',
            baseUrl: 'Base URL',
            model: 'Model',
            systemPrompt: 'System prompt',
            temperature: 'Temperature',
            maxTokens: 'Max tokens',
            save: 'Save',
            ragSaved: 'RAG config saved',
            ragSaveFailed: 'Failed to save RAG config',
            ragLoadFailed: 'Failed to load RAG config',
            index: 'Index',
            reloadIndex: 'Reload index',
            reloadOk: 'Index reloaded: {ts}',
            reloadFailed: 'Failed to reload index',
            coreUnavailable: 'Search backend unavailable (502)',
            toastSaved: 'Saved',
            accessOnly: 'Admin access only'
        }
    };

    function currentLang() {
        try {
            var l = window.localStorage.getItem(LANG_KEY);
            if (l === 'Русский' || l === 'English') return l;
        } catch (e) {}
        var appLang = (window.getCurrentLanguage && window.getCurrentLanguage()) || '';
        if (appLang === 'Русский' || appLang === 'English') return appLang;
        return 'English';
    }

    function t(key, u1, u2) {
        var s = (L[currentLang()] || L['English'])[key];
        if (s === undefined) s = L['English'][key] || key;
        if (u1 !== undefined) s = s.replace('{u}', u1);
        if (u2 !== undefined) s = s.replace('{r}', u2);
        return s;
    }

    function decodeJwtPayload(token) {
        if (!token) return null;
        try {
            var parts = token.split('.');
            if (parts.length !== 3) return null;
            var payload = parts[1].replace(/-/g, '+').replace(/_/g, '/');
            var pad = 4 - (payload.length % 4);
            if (pad < 4 && pad > 0) payload += '='.repeat(pad);
            var bin = atob(payload);
            var pct = bin.replace(/[\x80-\xff]/g, function (c) {
                return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
            });
            return JSON.parse(decodeURIComponent(pct));
        } catch (e) {
            return null;
        }
    }

    function isAdmin() {
        try {
            if (!window.ErroriumAPI || typeof window.ErroriumAPI.getTokens !== 'function') return false;
            var tokens = window.ErroriumAPI.getTokens();
            if (!tokens || !tokens.access_token) return false;
            var payload = decodeJwtPayload(tokens.access_token);
            return !!(payload && payload.role === 'admin');
        } catch (e) {
            return false;
        }
    }

    function getUserEmail() {
        try {
            var tokens = window.ErroriumAPI && window.ErroriumAPI.getTokens ? window.ErroriumAPI.getTokens() : null;
            var payload = tokens && tokens.access_token ? decodeJwtPayload(tokens.access_token) : null;
            return (payload && (payload.email || payload.username)) || '';
        } catch (e) {
            return '';
        }
    }

    /* ---------- toasts ---------- */

    var toastContainer = null;

    function ensureToastContainer() {
        if (toastContainer) return toastContainer;
        toastContainer = document.createElement('div');
        toastContainer.id = 'admin-toasts';
        toastContainer.className = 'admin-toasts';
        document.body.appendChild(toastContainer);
        return toastContainer;
    }

    function toast(message, kind) {
        var el = document.createElement('div');
        el.className = 'admin-toast' + (kind === 'error' ? ' admin-toast-error' : (kind === 'ok' ? ' admin-toast-ok' : ''));
        el.textContent = message;
        ensureToastContainer().appendChild(el);
        setTimeout(function () {
            if (el.parentNode) el.parentNode.removeChild(el);
        }, 3500);
    }

    /* ---------- helpers ---------- */

    function qs(sel, root) {
        return (root || document).querySelector(sel);
    }

    function on(el, ev, fn) {
        if (el) el.addEventListener(ev, fn);
    }

    function bytesToHuman(n) {
        n = Number(n) || 0;
        if (n < 1024) return n + ' B';
        if (n < 1048576) return (n / 1024).toFixed(1) + ' KB';
        return (n / 1048576).toFixed(2) + ' MB';
    }

    function formatTs(ts) {
        if (!ts) return '—';
        var d = new Date(ts);
        if (isNaN(d.getTime())) return String(ts);
        return d.toLocaleString(currentLang() === 'Русский' ? 'ru-RU' : 'en-US', {
            year: 'numeric', month: 'short', day: 'numeric',
            hour: '2-digit', minute: '2-digit'
        });
    }

    function showView() {
        var v = document.getElementById(VIEW_ID);
        if (v) v.style.display = 'flex';
    }

    function hideView() {
        var v = document.getElementById(VIEW_ID);
        if (v) v.style.display = 'none';
    }

    function apiRequest(method, path, body) {
        if (!window.ErroriumAPI || typeof window.ErroriumAPI.request !== 'function') {
            return Promise.reject({ status: 0, error: 'API client not loaded' });
        }
        return window.ErroriumAPI.request(method, path, body === undefined ? {} : { body: body }).then(function (res) {
            return res;
        });
    }

    function handleErr(err, fallback) {
        var msg = fallback;
        if (err && err.status === 403) msg = t('accessOnly');
        else if (err && err.status === 502) msg = t('coreUnavailable');
        else if (err && err.error) msg = String(err.error);
        toast(msg, 'error');
    }

    /* ---------- rendering ---------- */

    var activeTab = 'dashboard';
    var usersSearch = '';

    function renderHeader(root) {
        var header = document.createElement('div');
        header.className = 'admin-header';
        var h1 = document.createElement('h1');
        h1.className = 'admin-title';
        h1.textContent = t('title');
        var p = document.createElement('p');
        p.className = 'admin-subtitle';
        p.textContent = t('subtitle') + ': ' + (getUserEmail() || t('admin'));
        header.appendChild(h1);
        header.appendChild(p);
        root.appendChild(header);
    }

    function renderTabs(root) {
        var row = document.createElement('div');
        row.className = 'admin-tabs';
        var tabs = [
            { id: 'dashboard', label: t('tabDashboard') },
            { id: 'users', label: t('tabUsers') },
            { id: 'rag', label: t('tabRag') },
            { id: 'index', label: t('tabIndex') }
        ];
        tabs.forEach(function (tab) {
            var b = document.createElement('button');
            b.type = 'button';
            b.className = 'admin-tab' + (tab.id === activeTab ? ' admin-tab-active' : '');
            b.textContent = tab.label;
            b.setAttribute('data-tab', tab.id);
            on(b, 'click', function () {
                var btns = row.querySelectorAll('.admin-tab');
                for (var i = 0; i < btns.length; i++) btns[i].classList.remove('admin-tab-active');
                b.classList.add('admin-tab-active');
                activeTab = tab.id;
                renderContent(qs('#' + ROOT_ID + ' .admin-content'));
            });
            row.appendChild(b);
        });
        root.appendChild(row);
    }

    function renderContent(contentEl) {
        if (!contentEl) return;
        contentEl.innerHTML = '';
        if (activeTab === 'dashboard') renderDashboard(contentEl);
        else if (activeTab === 'users') renderUsers(contentEl);
        else if (activeTab === 'rag') renderRag(contentEl);
        else renderIndex(contentEl);
    }

    /* ---- Dashboard ---- */

    function renderDashboard(contentEl) {
        var grid = document.createElement('div');
        grid.className = 'admin-stats';
        var skeleton = document.createElement('div');
        skeleton.className = 'admin-skeleton';
        skeleton.textContent = t('loading');
        contentEl.appendChild(skeleton);

        apiRequest('GET', '/admin/stats').then(function (data) {
            grid.innerHTML = '';
            var stats = data || {};
            var lat = stats.latency_ms || {};
            var cards = [
                { label: t('documents'), value: String(stats.total_documents || 0) },
                { label: t('vocabulary'), value: String(stats.vocabulary_size || 0) },
                { label: t('avgDocLen'), value: String(Math.round(Number(stats.avg_doc_length || 0))) },
                { label: t('indexSize'), value: bytesToHuman(stats.index_size_bytes) },
                { label: t('coreReady'), value: stats.is_ready ? t('yes') : t('no'), badge: stats.is_ready ? 'ok' : 'error' },
                { label: t('latency') + ' p50/p95/p99', value: (Number(lat.p50) || 0) + ' / ' + (Number(lat.p95) || 0) + ' / ' + (Number(lat.p99) || 0) + ' ms' },
                { label: t('queriesTotal'), value: String(stats.queries_total || 0) }
            ];
            cards.forEach(function (c) {
                var card = document.createElement('div');
                card.className = 'admin-card';
                var lbl = document.createElement('div');
                lbl.className = 'admin-card-label';
                lbl.textContent = c.label;
                var val = document.createElement('div');
                val.className = 'admin-card-value' + (c.badge ? ' admin-badge admin-badge-' + c.badge : '');
                val.textContent = c.value;
                card.appendChild(lbl);
                card.appendChild(val);
                grid.appendChild(card);
            });
            contentEl.removeChild(skeleton);
            contentEl.appendChild(grid);
        }).catch(function (err) {
            contentEl.removeChild(skeleton);
            contentEl.appendChild(grid);
            handleErr(err, t('loadError'));
        });
    }

    /* ---- Users ---- */

    function renderUsers(contentEl) {
        var toolbar = document.createElement('div');
        toolbar.className = 'admin-toolbar';
        var input = document.createElement('input');
        input.type = 'text';
        input.className = 'admin-input';
        input.placeholder = t('searchPlaceholder');
        input.value = usersSearch;
        var loadBtn = document.createElement('button');
        loadBtn.type = 'button';
        loadBtn.className = 'admin-btn admin-btn-primary';
        loadBtn.textContent = t('load');
        on(loadBtn, 'click', function () {
            usersSearch = input.value || '';
            loadUsers(contentEl);
        });
        on(input, 'keydown', function (e) {
            if (e.key === 'Enter') { usersSearch = input.value || ''; loadUsers(contentEl); }
        });
        toolbar.appendChild(input);
        toolbar.appendChild(loadBtn);
        contentEl.appendChild(toolbar);

        var tableWrap = document.createElement('div');
        tableWrap.className = 'admin-table-wrap';
        var table = document.createElement('table');
        table.className = 'admin-table';
        var thead = document.createElement('thead');
        var tr = document.createElement('tr');
        ['username', 'email', 'role', 'active', 'lastLogin', ''].forEach(function (k) {
            var th = document.createElement('th');
            th.textContent = k === '' ? t('actions') : t(k);
            tr.appendChild(th);
        });
        thead.appendChild(tr);
        table.appendChild(thead);
        var tbody = document.createElement('tbody');
        table.appendChild(tbody);
        tableWrap.appendChild(table);
        contentEl.appendChild(tableWrap);

        if (usersSearch !== '') loadUsers(contentEl);
    }

    function loadUsers(contentEl) {
        var tbody = qs('#' + ROOT_ID + ' .admin-table tbody');
        if (!tbody) return;
        tbody.innerHTML = '';
        var q = usersSearch;
        var path = '/admin/users' + (q ? ('?search=' + encodeURIComponent(q)) : '');
        apiRequest('GET', path).then(function (data) {
            renderUsersRows(tbody, (data && data.items) || []);
        }).catch(function (err) {
            handleErr(err, t('usersLoadFailed'));
        });
    }

    function renderUsersRows(tbody, items) {
        tbody.innerHTML = '';
        items.forEach(function (u) {
            var tr = document.createElement('tr');
            var tdUser = document.createElement('td');
            tdUser.textContent = u.username || u.email || '';
            var tdEmail = document.createElement('td');
            tdEmail.textContent = u.email || '';
            var tdRole = document.createElement('td');
            var roleBadge = document.createElement('span');
            roleBadge.className = 'admin-badge' + (u.role === 'admin' ? ' admin-badge-admin' : ' admin-badge-user');
            roleBadge.textContent = u.role === 'admin' ? 'admin' : 'user';
            tdRole.appendChild(roleBadge);
            var tdActive = document.createElement('td');
            var actBadge = document.createElement('span');
            actBadge.className = 'admin-badge' + (u.is_active ? ' admin-badge-ok' : ' admin-badge-error');
            actBadge.textContent = u.is_active ? t('yes') : t('no');
            tdActive.appendChild(actBadge);
            var tdLogin = document.createElement('td');
            tdLogin.textContent = formatTs(u.last_login_at || u.created_at);

            var tdActions = document.createElement('td');
            tdActions.className = 'admin-actions';

            var roleBtn = document.createElement('button');
            roleBtn.type = 'button';
            roleBtn.className = 'admin-btn admin-btn-outline admin-sm';
            roleBtn.textContent = u.role === 'admin' ? t('makeUser') : t('makeAdmin');
            on(roleBtn, 'click', function () {
                var targetRole = u.role === 'admin' ? 'user' : 'admin';
                if (!window.confirm(t('confirmRole', u.username || u.id, targetRole))) return;
                roleBtn.disabled = true;
                apiRequest('PATCH', '/admin/users/' + encodeURIComponent(u.id), { role: targetRole }).then(function (res) {
                    u.role = (res && res.role) || targetRole;
                    renderUsersRows(tbody, items);
                    toast(t('roleChanged'), 'ok');
                }).catch(function (err) {
                    handleErr(err, t('roleChangeFailed'));
                    roleBtn.disabled = false;
                });
            });

            var actBtn = document.createElement('button');
            actBtn.type = 'button';
            actBtn.className = 'admin-btn admin-btn-danger admin-sm';
            actBtn.textContent = u.is_active ? t('block') : t('unblock');
            on(actBtn, 'click', function () {
                var nextActive = !u.is_active;
                actBtn.disabled = true;
                apiRequest('PATCH', '/admin/users/' + encodeURIComponent(u.id), { is_active: nextActive }).then(function (res) {
                    u.is_active = res && res.is_active !== undefined ? res.is_active : nextActive;
                    renderUsersRows(tbody, items);
                    toast(nextActive ? t('unblocked') : t('blocked'), 'ok');
                }).catch(function (err) {
                    handleErr(err, t('blockFailed'));
                    actBtn.disabled = false;
                });
            });

            tdActions.appendChild(roleBtn);
            tdActions.appendChild(actBtn);
            tr.appendChild(tdUser);
            tr.appendChild(tdEmail);
            tr.appendChild(tdRole);
            tr.appendChild(tdActive);
            tr.appendChild(tdLogin);
            tr.appendChild(tdActions);
            tbody.appendChild(tr);
        });
    }

    /* ---- RAG config ---- */

    function renderRag(contentEl) {
        var form = document.createElement('div');
        form.className = 'admin-form';
        contentEl.appendChild(form);

        var fields = {
            provider: { type: 'select', options: ['openai', 'gigachat', 'stub'], label: 'provider' },
            base_url: { type: 'text', label: 'baseUrl' },
            model: { type: 'text', label: 'model' },
            system_prompt: { type: 'textarea', label: 'systemPrompt' },
            temperature: { type: 'number', min: 0, max: 2, step: 0.1, label: 'temperature' },
            max_tokens: { type: 'number', min: 1, label: 'maxTokens' }
        };

        var inputs = {};

        Object.keys(fields).forEach(function (key) {
            var f = fields[key];
            var row = document.createElement('div');
            row.className = 'admin-form-row';
            var label = document.createElement('label');
            label.className = 'admin-form-label';
            label.textContent = t(f.label) || key;
            label.setAttribute('for', 'admin-rag-' + key);
            var el;
            if (f.type === 'select') {
                el = document.createElement('select');
                el.id = 'admin-rag-' + key;
                el.className = 'admin-input';
                f.options.forEach(function (opt) {
                    var o = document.createElement('option');
                    o.value = opt;
                    o.textContent = opt;
                    el.appendChild(o);
                });
            } else if (f.type === 'textarea') {
                el = document.createElement('textarea');
                el.id = 'admin-rag-' + key;
                el.className = 'admin-input admin-textarea';
                el.rows = 4;
            } else {
                el = document.createElement('input');
                el.id = 'admin-rag-' + key;
                el.type = f.type;
                el.className = 'admin-input';
                if (f.min !== undefined) el.min = f.min;
                if (f.max !== undefined) el.max = f.max;
                if (f.step !== undefined) el.step = f.step;
            }
            el.dataset.field = key;
            inputs[key] = el;
            row.appendChild(label);
            row.appendChild(el);
            form.appendChild(row);
        });

        var saveBtn = document.createElement('button');
        saveBtn.type = 'button';
        saveBtn.className = 'admin-btn admin-btn-primary';
        saveBtn.textContent = t('save');
        on(saveBtn, 'click', function () {
            var body = {};
            Object.keys(fields).forEach(function (key) {
                var v = inputs[key].value;
                if (fields[key].type === 'number') v = Number(v);
                body[key] = v;
            });
            saveBtn.disabled = true;
            apiRequest('PUT', '/admin/rag/config', body).then(function () {
                toast(t('ragSaved'), 'ok');
                saveBtn.disabled = false;
            }).catch(function (err) {
                handleErr(err, t('ragSaveFailed'));
                saveBtn.disabled = false;
            });
        });
        form.appendChild(saveBtn);

        apiRequest('GET', '/admin/rag/config').then(function (data) {
            if (!data) return;
            Object.keys(fields).forEach(function (key) {
                if (data[key] !== undefined && inputs[key]) inputs[key].value = data[key];
            });
        }).catch(function (err) {
            handleErr(err, t('ragLoadFailed'));
        });
    }

    /* ---- Index reload ---- */

    function renderIndex(contentEl) {
        var wrap = document.createElement('div');
        wrap.className = 'admin-index';
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'admin-btn admin-btn-primary';
        btn.textContent = t('reloadIndex');
        on(btn, 'click', function () {
            btn.disabled = true;
            apiRequest('POST', '/admin/index/reload').then(function (res) {
                toast(t('reloadOk', String(res && res.build_timestamp || '')), 'ok');
                btn.disabled = false;
            }).catch(function (err) {
                handleErr(err, t('reloadFailed'));
                btn.disabled = false;
            });
        });
        wrap.appendChild(btn);
        contentEl.appendChild(wrap);
    }

    /* ---------- public show ---------- */

    function show() {
        if (!isAdmin()) return;
        var root = document.getElementById(ROOT_ID);
        if (!root) return;
        showView();
        root.innerHTML = '';
        renderHeader(root);
        renderTabs(root);
        var content = document.createElement('div');
        content.className = 'admin-content';
        root.appendChild(content);
        renderContent(content);
    }

    function navigate(path) {
        if (window.ErroriumNavigate && typeof window.ErroriumNavigate === 'function') {
            window.ErroriumNavigate(path);
        } else {
            if (window.location.pathname !== path) window.history.pushState({}, '', path);
            if (typeof renderCurrentRoute === 'function') renderCurrentRoute();
        }
    }

    /* ---------- init ---------- */

    function init() {
        if (!isAdmin()) return;
        var links = document.querySelector(SIDEBAR_LINKS);
        if (links && !document.getElementById('btn-admin')) {
            var btn = document.createElement('button');
            btn.id = 'btn-admin';
            btn.className = 'sidebar-link-btn';
            btn.title = 'Admin panel';
            var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            svg.setAttribute('width', '18');
            svg.setAttribute('height', '18');
            svg.setAttribute('viewBox', '0 0 24 24');
            svg.setAttribute('fill', 'none');
            svg.setAttribute('stroke', 'currentColor');
            svg.setAttribute('stroke-width', '2');
            svg.setAttribute('stroke-linecap', 'round');
            svg.setAttribute('stroke-linejoin', 'round');
            var paths = [
                'M12 1v3m0 16v3',
                'M4.22 4.22l2.12 2.12m6.36 6.36l2.12 2.12',
                'M1 12h3m16 0h3',
                'M4.22 19.78l2.12-2.12m6.36-6.36l2.12-2.12'
            ];
            paths.forEach(function (d) {
                var p = document.createElementNS('http://www.w3.org/2000/svg', 'path');
                p.setAttribute('d', d);
                svg.appendChild(p);
            });
            btn.appendChild(svg);
            var span = document.createElement('span');
            span.textContent = t('admin');
            btn.appendChild(span);
            on(btn, 'click', function () {
                navigate('/admin');
            });
            links.appendChild(btn);
        }

        if (window.location.pathname.indexOf('/admin') === 0) {
            navigate('/admin');
        }

        window.addEventListener('popstate', function () {
            if (window.location.pathname.indexOf('/admin') !== 0) hideView();
        });

        window.addEventListener('click', function (e) {
            if (!e.target) return;
            if (e.target.closest('#' + ROOT_ID)) return;
            if (e.target.closest('#btn-admin')) return;
            if (window.location.pathname.indexOf('/admin') !== 0) hideView();
        });
    }

    window.ErroriumAdmin = {
        isAdmin: isAdmin,
        show: show,
        navigate: navigate
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
