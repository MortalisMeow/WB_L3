let currentPage = 1;
let limit = 10;
let totalPages = 1;

function loadComments() {
    fetch(`/comments?page=${currentPage}&limit=${limit}`)
        .then(res => res.json())
        .then(data => {
            renderPage(data);
        })
        .catch(err => console.error(err));
}

function renderPage(data) {
    const container = document.getElementById('comments-tree');
    container.innerHTML = '';

    if (data.comments && data.comments.length > 0) {
        data.comments.forEach(comment => {
            container.appendChild(renderComment(comment, 0));
        });
    }

    totalPages = Math.ceil(data.total / limit) || 1;
    document.getElementById('page-info').textContent = `Страница ${data.page} из ${totalPages}`;
    document.getElementById('prev-btn').disabled = data.page <= 1;
    document.getElementById('next-btn').disabled = data.page >= totalPages;
}

function renderComment(comment, level) {
    const div = document.createElement('div');
    div.className = 'comment';
    div.style.marginLeft = level * 30 + 'px';

    const header = document.createElement('div');
    header.className = 'comment-header';
    header.innerHTML = `<strong>${escapeHtml(comment.author)}</strong> — ${formatDate(comment.created_at)}`;
    div.appendChild(header);

    const body = document.createElement('div');
    body.className = 'comment-body';
    body.innerHTML = comment.matched ? `<span class="highlight">${escapeHtml(comment.body)}</span>` : escapeHtml(comment.body);
    div.appendChild(body);

    const actions = document.createElement('div');
    actions.className = 'comment-actions';

    const replyBtn = document.createElement('button');
    replyBtn.textContent = 'Ответить';
    replyBtn.onclick = function() {
        showReplyForm(div, comment.id);
    };
    actions.appendChild(replyBtn);

    const deleteBtn = document.createElement('button');
    deleteBtn.textContent = 'Удалить';
    deleteBtn.onclick = function() {
        if (confirm('Удалить комментарий и все ответы?')) {
            deleteComment(comment.id);
        }
    };
    actions.appendChild(deleteBtn);

    div.appendChild(actions);

    if (comment.children && comment.children.length > 0) {
        comment.children.forEach(child => {
            div.appendChild(renderComment(child, level + 1));
        });
    }

    return div;
}

function showReplyForm(parentDiv, parentId) {
    const existingForm = parentDiv.querySelector('.reply-form');
    if (existingForm) {
        existingForm.remove();
        return;
    }

    const form = document.createElement('div');
    form.className = 'reply-form';

    const textarea = document.createElement('textarea');
    textarea.placeholder = 'Ваш ответ...';
    form.appendChild(textarea);

    const authorInput = document.createElement('input');
    authorInput.type = 'text';
    authorInput.placeholder = 'Ваше имя';
    form.appendChild(authorInput);

    const submitBtn = document.createElement('button');
    submitBtn.textContent = 'Отправить';
    submitBtn.onclick = function() {
        createComment(parentId, textarea.value, authorInput.value);
    };
    form.appendChild(submitBtn);

    parentDiv.appendChild(form);
}

function createComment(parentId, body, author) {
    if (!body) {
        body = document.getElementById('new-comment-body').value;
        author = document.getElementById('new-comment-author').value;
    }

    if (!body.trim()) {
        alert('Введите текст комментария');
        return;
    }

    const data = {
        body: body,
        author: author || 'Anonymous'
    };

    if (parentId) {
        data.parent_id = parentId;
    }

    fetch('/comments', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
    })
        .then(res => res.json())
        .then(() => {
            if (!parentId) {
                document.getElementById('new-comment-body').value = '';
                document.getElementById('new-comment-author').value = '';
            }
            loadComments();
        })
        .catch(err => console.error(err));
}

function deleteComment(id) {
    fetch(`/comments/${id}`, {
        method: 'DELETE'
    })
        .then(() => loadComments())
        .catch(err => console.error(err));
}

function searchComments() {
    const query = document.getElementById('search-input').value.trim();
    if (!query) {
        loadComments();
        return;
    }

    fetch(`/search?q=${encodeURIComponent(query)}`)
        .then(res => res.json())
        .then(data => {
            const container = document.getElementById('comments-tree');
            container.innerHTML = '';

            if (data && data.length > 0) {
                data.forEach(comment => {
                    container.appendChild(renderComment(comment, 0));
                });
            } else {
                container.innerHTML = '<p>Ничего не найдено</p>';
            }

            document.getElementById('page-info').textContent = 'Результаты поиска';
            document.getElementById('prev-btn').disabled = true;
            document.getElementById('next-btn').disabled = true;
        })
        .catch(err => console.error(err));
}

function clearSearch() {
    document.getElementById('search-input').value = '';
    currentPage = 1;
    loadComments();
}

function changePage(delta) {
    currentPage += delta;
    if (currentPage < 1) currentPage = 1;
    if (currentPage > totalPages) currentPage = totalPages;
    loadComments();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU');
}

loadComments();