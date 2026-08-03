var state = {
    token: localStorage.getItem('wc_token'),
    username: localStorage.getItem('wc_username'),
    role: localStorage.getItem('wc_role')
};

function authHeaders() {
    return {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + state.token
    };
}

function canWrite() {
    return state.role === 'admin' || state.role === 'manager';
}

function canDelete() {
    return state.role === 'admin';
}

function canExport() {
    return state.role === 'admin' || state.role === 'manager';
}

function roleLabel(role) {
    if (role === 'admin') return 'Админ';
    if (role === 'manager') return 'Менеджер';
    return 'Наблюдатель';
}

function login() {
    var role = document.getElementById('role-select').value;
    fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: role, role: role })
    }).then(function(res) {
        return res.json().then(function(data) {
            if (!res.ok) throw new Error(data.error || 'login failed');
            state.token = data.token;
            state.username = data.username;
            state.role = data.role;
            localStorage.setItem('wc_token', data.token);
            localStorage.setItem('wc_username', data.username);
            localStorage.setItem('wc_role', data.role);
            showApp();
        });
    }).catch(function(err) {
        alert(err.message);
    });
}

function logout() {
    state.token = null;
    state.username = null;
    state.role = null;
    localStorage.removeItem('wc_token');
    localStorage.removeItem('wc_username');
    localStorage.removeItem('wc_role');
    document.getElementById('login-section').classList.remove('hidden');
    document.getElementById('user-section').classList.add('hidden');
    document.getElementById('create-section').classList.add('hidden');
    document.getElementById('items-section').classList.add('hidden');
    document.getElementById('history-section').classList.add('hidden');
}

function showApp() {
    document.getElementById('login-section').classList.add('hidden');
    document.getElementById('user-section').classList.remove('hidden');
    document.getElementById('items-section').classList.remove('hidden');
    document.getElementById('history-section').classList.remove('hidden');
    document.getElementById('user-info').textContent =
        state.username + ' (' + roleLabel(state.role) + ')';

    if (canWrite()) {
        document.getElementById('create-section').classList.remove('hidden');
    } else {
        document.getElementById('create-section').classList.add('hidden');
    }

    if (canExport()) {
        document.getElementById('csv-link').classList.remove('hidden');
    } else {
        document.getElementById('csv-link').classList.add('hidden');
    }

    loadItems();
    loadHistory();
}

function api(url, opts) {
    opts = opts || {};
    opts.headers = opts.headers || authHeaders();
    return fetch(url, opts).then(function(res) {
        if (res.status === 401) {
            logout();
            throw new Error('session expired');
        }
        return res;
    });
}

function createItem() {
    var body = {
        name: document.getElementById('name').value,
        sku: document.getElementById('sku').value,
        quantity: parseInt(document.getElementById('quantity').value, 10) || 0,
        price: parseFloat(document.getElementById('price').value) || 0,
        description: document.getElementById('description').value
    };
    api('/items', {
        method: 'POST',
        body: JSON.stringify(body)
    }).then(function(res) {
        if (!res.ok) return res.json().then(function(e) { alert(e.error); });
        document.getElementById('name').value = '';
        document.getElementById('sku').value = '';
        document.getElementById('quantity').value = '0';
        document.getElementById('price').value = '0';
        document.getElementById('description').value = '';
        loadItems();
        loadHistory();
    });
}

function loadItems() {
    api('/items')
        .then(function(res) { return res.json(); })
        .then(function(items) {
            var tbody = document.querySelector('#items-table tbody');
            tbody.innerHTML = '';
            items.forEach(function(item) {
                var tr = document.createElement('tr');
                var actions = '<button class="hist-btn" onclick="showItemHistory(' + item.id + ')">История</button>';
                if (canWrite()) {
                    actions = '<button class="edit-btn" onclick="editItem(' + item.id + ')">Изменить</button>' + actions;
                }
                if (canDelete()) {
                    actions += '<button class="del-btn" onclick="deleteItem(' + item.id + ')">Удалить</button>';
                }
                tr.innerHTML =
                    '<td>' + item.id + '</td>' +
                    '<td>' + escapeHtml(item.name) + '</td>' +
                    '<td>' + escapeHtml(item.sku) + '</td>' +
                    '<td>' + item.quantity + '</td>' +
                    '<td>' + item.price.toFixed(2) + '</td>' +
                    '<td>' + escapeHtml(item.description) + '</td>' +
                    '<td>' + actions + '</td>';
                tbody.appendChild(tr);
            });
        });
}

function editItem(id) {
    api('/items/' + id)
        .then(function(res) { return res.json(); })
        .then(function(item) {
            var name = prompt('Название', item.name);
            if (name === null) return;
            var sku = prompt('SKU', item.sku);
            if (sku === null) return;
            var qty = prompt('Кол-во', item.quantity);
            if (qty === null) return;
            var price = prompt('Цена', item.price);
            if (price === null) return;
            var desc = prompt('Описание', item.description);
            if (desc === null) return;
            api('/items/' + id, {
                method: 'PUT',
                body: JSON.stringify({
                    name: name,
                    sku: sku,
                    quantity: parseInt(qty, 10) || 0,
                    price: parseFloat(price) || 0,
                    description: desc
                })
            }).then(function(res) {
                if (!res.ok) return res.json().then(function(e) { alert(e.error); });
                loadItems();
                loadHistory();
            });
        });
}

function deleteItem(id) {
    if (!confirm('Удалить товар #' + id + '?')) return;
    api('/items/' + id, { method: 'DELETE' }).then(function(res) {
        if (!res.ok) return res.json().then(function(e) { alert(e.error); });
        loadItems();
        loadHistory();
    });
}

function historyQuery() {
    var parts = [];
    var from = document.getElementById('hist-from').value;
    var to = document.getElementById('hist-to').value;
    var user = document.getElementById('hist-user').value;
    var action = document.getElementById('hist-action').value;
    var itemId = document.getElementById('hist-item-id').value;
    if (from) parts.push('from=' + from);
    if (to) parts.push('to=' + to);
    if (user) parts.push('user=' + encodeURIComponent(user));
    if (action) parts.push('action=' + action);
    if (itemId) parts.push('item_id=' + itemId);
    return parts.length ? '?' + parts.join('&') : '';
}

function loadHistory() {
    var qs = historyQuery();
    if (canExport()) {
        document.getElementById('csv-link').href = '/history/export' + qs + (qs ? '&' : '?') + 'token=';
    }
    api('/history' + qs)
        .then(function(res) { return res.json(); })
        .then(function(entries) {
            renderHistory(entries);
        });
}

function renderHistory(entries) {
    var tbody = document.querySelector('#history-table tbody');
    tbody.innerHTML = '';
    entries.forEach(function(e) {
        var tr = document.createElement('tr');
        var cls = 'action-' + e.action.toLowerCase();
        tr.innerHTML =
            '<td>' + e.id + '</td>' +
            '<td>' + e.item_id + '</td>' +
            '<td class="' + cls + '">' + e.action + '</td>' +
            '<td>' + escapeHtml(e.changed_by) + '</td>' +
            '<td>' + formatDate(e.changed_at) + '</td>' +
            '<td><button class="hist-btn" onclick="showDiff(' + e.id + ')">Diff</button></td>';
        tbody.appendChild(tr);
    });
}

function showItemHistory(itemId) {
    document.getElementById('hist-item-id').value = itemId;
    loadHistory();
    document.getElementById('history-section').scrollIntoView({ behavior: 'smooth' });
}

function showDiff(historyId) {
    api('/history/' + historyId + '/diff')
        .then(function(res) { return res.json(); })
        .then(function(data) {
            var html = '<p><strong>Действие:</strong> ' + data.entry.action +
                ' | <strong>Кто:</strong> ' + escapeHtml(data.entry.changed_by) +
                ' | <strong>Когда:</strong> ' + formatDate(data.entry.changed_at) + '</p>';
            var keys = Object.keys(data.changes);
            if (keys.length === 0) {
                html += '<p>Нет отличий</p>';
            } else {
                html += '<table class="diff-table"><thead><tr><th>Поле</th><th>Было</th><th>Стало</th></tr></thead><tbody>';
                keys.forEach(function(k) {
                    html += '<tr><td>' + k + '</td>' +
                        '<td class="diff-old">' + escapeHtml(String(data.changes[k].old != null ? data.changes[k].old : '')) + '</td>' +
                        '<td class="diff-new">' + escapeHtml(String(data.changes[k].new != null ? data.changes[k].new : '')) + '</td></tr>';
                });
                html += '</tbody></table>';
            }
            document.getElementById('modal-title').textContent = 'Изменения #' + historyId;
            document.getElementById('modal-body').innerHTML = html;
            document.getElementById('modal').classList.remove('hidden');
        });
}

function closeModal() {
    document.getElementById('modal').classList.add('hidden');
}

function formatDate(s) {
    return new Date(s).toLocaleString('ru-RU');
}

function escapeHtml(s) {
    if (!s) return '';
    return String(s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

document.getElementById('csv-link').addEventListener('click', function(e) {
    e.preventDefault();
    if (!canExport()) return;
    var qs = historyQuery();
    fetch('/history/export' + qs, { headers: { 'Authorization': 'Bearer ' + state.token } })
        .then(function(res) { return res.blob(); })
        .then(function(blob) {
            var url = URL.createObjectURL(blob);
            var a = document.createElement('a');
            a.href = url;
            a.download = 'history.csv';
            a.click();
            URL.revokeObjectURL(url);
        });
});

if (state.token) {
    showApp();
}
