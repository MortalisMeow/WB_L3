document.getElementById('upload-form').addEventListener('submit', function(e) {
    e.preventDefault();
    var input = document.getElementById('file-input');
    var file = input.files[0];
    if (!file) return;

    var formData = new FormData();
    formData.append('image', file);

    fetch('/upload', { method: 'POST', body: formData })
        .then(function() {
            input.value = '';
            loadImages();
        });
});

function loadImages() {
    fetch('/images')
        .then(function(res) { return res.json(); })
        .then(function(images) {
            var container = document.getElementById('images-list');
            container.innerHTML = '';

            images.forEach(function(img) {
                var card = document.createElement('div');
                card.className = 'image-card';

                if (img.status === 'done') {
                    var thumb = document.createElement('img');
                    thumb.src = '/image/' + img.id;
                    card.appendChild(thumb);
                } else {
                    var placeholder = document.createElement('div');
                    placeholder.className = 'placeholder';
                    placeholder.textContent = img.status;
                    card.appendChild(placeholder);
                }

                var info = document.createElement('div');
                info.innerHTML = '<div>' + img.filename + '</div><span class="status">' + img.status + '</span>';
                card.appendChild(info);

                var del = document.createElement('button');
                del.textContent = 'X';
                del.onclick = function() {
                    fetch('/image/' + img.id, { method: 'DELETE' }).then(loadImages);
                };
                card.appendChild(del);

                container.appendChild(card);
            });
        });
}

setInterval(loadImages, 3000);
loadImages();