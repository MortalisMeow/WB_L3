function create() {
    var name = document.getElementById('name').value;
    var date = document.getElementById('date').value;
    var seats = document.getElementById('seats').value;

    fetch('/events', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name, date: date, seats: parseInt(seats) })
    }).then(function() {
        document.getElementById('name').value = '';
        document.getElementById('date').value = '';
        document.getElementById('seats').value = '';
        load();
    });
}

function load() {
    fetch('/events')
        .then(function(res) { return res.json(); })
        .then(function(events) {
            var list = document.getElementById('list');
            list.innerHTML = '';

            events.forEach(function(e) {
                var card = document.createElement('div');
                card.className = 'card';
                card.innerHTML = '<h3>' + e.name + '</h3>' +
                    '<p>Дата: ' + e.date + '</p>' +
                    '<p>Мест: ' + e.booked + ' / ' + e.seats + '</p>' +
                    '<div id="b-' + e.id + '"></div>' +
                    '<input type="text" id="u-' + e.id + '" placeholder="Имя">' +
                    '<button onclick="book(' + e.id + ')">Забронировать</button>' +
                    '<button onclick="loadBookings(' + e.id + ')">Обновить</button>' +
                    '<button onclick="del(' + e.id + ')">Удалить</button>';
                list.appendChild(card);
            });
        });
}

function book(id) {
    var user = document.getElementById('u-' + id).value;

    fetch('/events/' + id + '/book', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_name: user })
    }).then(function() {
        load();
        loadBookings(id);
    });
}

function loadBookings(id) {
    fetch('/events/' + id)
        .then(function(res) { return res.json(); })
        .then(function(data) {
            var box = document.getElementById('b-' + id);
            box.innerHTML = '';

            data.bookings.forEach(function(b) {
                var div = document.createElement('div');
                div.className = b.status;
                div.innerHTML = b.user_name + ' - ' + b.status;

                if (b.status === 'pending') {
                    var btn = document.createElement('button');
                    btn.textContent = 'Оплатить';
                    btn.onclick = function() {
                        fetch('/events/' + id + '/confirm', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ booking_id: b.id })
                        }).then(function() {
                            load();
                            loadBookings(id);
                        });
                    };
                    div.appendChild(btn);
                }

                box.appendChild(div);
            });
        });
}

function del(id) {
    fetch('/events/' + id, { method: 'DELETE' }).then(load);
}

setInterval(load, 5000);
load();