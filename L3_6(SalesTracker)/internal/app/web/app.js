var chart;

function today() {
    return new Date().toISOString().slice(0, 10);
}

function monthAgo() {
    var d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().slice(0, 10);
}

function filters() {
    return {
        from: document.getElementById('filter-from').value,
        to: document.getElementById('filter-to').value,
        type: document.getElementById('filter-type').value,
        category: document.getElementById('filter-category').value
    };
}

function queryString(f) {
    var parts = [];
    if (f.from) parts.push('from=' + f.from);
    if (f.to) parts.push('to=' + f.to);
    if (f.type) parts.push('type=' + encodeURIComponent(f.type));
    if (f.category) parts.push('category=' + encodeURIComponent(f.category));
    return parts.length ? '?' + parts.join('&') : '';
}

function createItem() {
    var body = {
        type: document.getElementById('type').value,
        amount: parseFloat(document.getElementById('amount').value),
        category: document.getElementById('category').value,
        occurred_at: document.getElementById('date').value
    };

    fetch('/items', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
    }).then(function(res) {
        if (!res.ok) return res.json().then(function(e) { alert(e.error); });
        document.getElementById('amount').value = '';
        document.getElementById('category').value = '';
        loadAll();
    });
}

function loadItems() {
    var qs = queryString(filters());
    fetch('/items' + qs)
        .then(function(res) { return res.json(); })
        .then(function(items) {
            var tbody = document.querySelector('#items-table tbody');
            tbody.innerHTML = '';
            items.forEach(function(item) {
                var tr = document.createElement('tr');
                var date = item.occurred_at.slice(0, 10);
                tr.innerHTML =
                    '<td>' + item.id + '</td>' +
                    '<td class="' + item.type + '">' + (item.type === 'income' ? 'Доход' : 'Расход') + '</td>' +
                    '<td>' + item.amount.toFixed(2) + '</td>' +
                    '<td>' + item.category + '</td>' +
                    '<td>' + date + '</td>' +
                    '<td><button class="del-btn" onclick="deleteItem(' + item.id + ')">Удалить</button></td>';
                tbody.appendChild(tr);
            });
        });
}

function deleteItem(id) {
    fetch('/items/' + id, { method: 'DELETE' }).then(loadAll);
}

function loadAnalytics() {
    var qs = queryString(filters());
    fetch('/analytics' + qs)
        .then(function(res) { return res.json(); })
        .then(function(data) {
            var s = data.stats;
            document.getElementById('stats').innerHTML =
                statBox('Кол-во', s.count) +
                statBox('Сумма', fmt(s.sum)) +
                statBox('Среднее', fmt(s.avg)) +
                statBox('Медиана', fmt(s.median)) +
                statBox('90 перц.', fmt(s.percentile_90)) +
                statBox('Доходы', fmt(s.income_sum)) +
                statBox('Расходы', fmt(s.expense_sum));

            var labels = data.daily.map(function(d) { return d.date; });
            var values = data.daily.map(function(d) { return d.total; });

            if (chart) chart.destroy();
            chart = new Chart(document.getElementById('chart'), {
                type: 'bar',
                data: {
                    labels: labels,
                    datasets: [{
                        label: 'Баланс за день',
                        data: values,
                        backgroundColor: values.map(function(v) {
                            return v >= 0 ? '#16a34a' : '#dc2626';
                        })
                    }]
                },
                options: {
                    responsive: true,
                    scales: { y: { beginAtZero: true } }
                }
            });
        });
}

function statBox(label, value) {
    return '<div class="stat-box"><div class="label">' + label + '</div><div class="value">' + value + '</div></div>';
}

function fmt(n) {
    return Number(n).toFixed(2);
}

function updateCsvLink() {
    document.getElementById('csv-link').href = '/items/export' + queryString(filters());
}

function loadAll() {
    updateCsvLink();
    loadItems();
    loadAnalytics();
}

document.getElementById('date').value = today();
document.getElementById('filter-from').value = monthAgo();
document.getElementById('filter-to').value = today();
loadAll();
