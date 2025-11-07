let currentPage = 1;
let perPage = 20;
let totalPages = 1;
let selectedUsers = new Set();
let sortOrder = 'DESC'; // По умолчанию DESC

// Загрузка данных при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    loadUsers(currentPage);
});

// Переключение сортировки
function toggleSort() {
    sortOrder = sortOrder === 'DESC' ? 'ASC' : 'DESC';
    updateSortIcon();
    loadUsers(1); // Загружаем первую страницу с новой сортировкой
}

// Обновление иконки сортировки
function updateSortIcon() {
    const icon = document.getElementById('sort-icon');
    if (sortOrder === 'DESC') {
        icon.className = 'bi bi-arrow-down';
    } else {
        icon.className = 'bi bi-arrow-up';
    }
}

// Загрузка пользователей
async function loadUsers(page) {
    currentPage = page;

    // Показываем индикатор загрузки
    document.getElementById('loading').style.display = 'block';
    document.getElementById('table-wrapper').style.display = 'none';
    document.getElementById('error-message').style.display = 'none';

    try {
        const response = await fetch(`/api/users?page=${page}&per_page=${perPage}&sort_order=${sortOrder}`);
        if (!response.ok) {
            throw new Error('Ошибка загрузки данных');
        }

        const data = await response.json();

        // Обновляем статистику
        document.getElementById('total-count').textContent = data.total;
        document.getElementById('current-page').textContent = data.page;
        document.getElementById('per-page').textContent = data.per_page;
        document.getElementById('total-pages').textContent = data.total_pages;

        // Обновляем сортировку из ответа
        if (data.sort_order) {
            sortOrder = data.sort_order;
            updateSortIcon();
        }

        totalPages = data.total_pages;

        // Рендерим таблицу
        renderTable(data.data);

        // Рендерим пагинацию
        renderPagination(data.page, data.total_pages);

        // Скрываем загрузку и показываем таблицу
        document.getElementById('loading').style.display = 'none';
        document.getElementById('table-wrapper').style.display = 'block';
    } catch (error) {
        console.error('Ошибка:', error);
        document.getElementById('loading').style.display = 'none';
        document.getElementById('error-message').textContent = 'Ошибка загрузки данных: ' + error.message;
        document.getElementById('error-message').style.display = 'block';
    }
}

// Рендеринг таблицы
function renderTable(users) {
    const tbody = document.getElementById('users-table-body');
    tbody.innerHTML = '';

    users.forEach(user => {
        const row = document.createElement('tr');

        // Чекбокс
        const checkboxCell = document.createElement('td');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = user.user_id;
        checkbox.checked = selectedUsers.has(user.user_id);
        checkbox.onchange = () => toggleUserSelection(user.user_id);
        checkboxCell.appendChild(checkbox);

        // User ID
        const userIdCell = document.createElement('td');
        userIdCell.textContent = user.user_id;

        // Citizenship ID
        const citizenshipCell = document.createElement('td');
        const citizenshipBadge = document.createElement('span');
        citizenshipBadge.className = 'badge bg-primary badge-custom';
        citizenshipBadge.textContent = user.citizenship_id || 'N/A';
        citizenshipCell.appendChild(citizenshipBadge);

        // Document Files
        const docFilesCell = document.createElement('td');
        if (user.document_files && user.document_files.trim() !== '') {
            const link = document.createElement('a');
            link.href = user.document_files;
            link.target = '_blank';
            link.className = 'file-link';
            link.title = user.document_files;
            link.innerHTML = '<i class="bi bi-link-45deg"></i> Ссылка';
            docFilesCell.appendChild(link);
        } else {
            docFilesCell.innerHTML = '<span class="text-muted">-</span>';
        }

        // Status Document
        const docStatusCell = document.createElement('td');
        const docBadge = document.createElement('span');
        docBadge.className = user.document ? 'badge bg-success badge-custom' : 'badge bg-secondary badge-custom';
        docBadge.innerHTML = user.document ? '<i class="bi bi-check-circle"></i> True' : '<i class="bi bi-x-circle"></i> False';
        docStatusCell.appendChild(docBadge);

        // Address Files
        const addrFilesCell = document.createElement('td');
        if (user.address_files && user.address_files.trim() !== '') {
            const link = document.createElement('a');
            link.href = user.address_files;
            link.target = '_blank';
            link.className = 'file-link';
            link.title = user.address_files;
            link.innerHTML = '<i class="bi bi-link-45deg"></i> Ссылка';
            addrFilesCell.appendChild(link);
        } else {
            addrFilesCell.innerHTML = '<span class="text-muted">-</span>';
        }

        // Status Address
        const addrStatusCell = document.createElement('td');
        const addrBadge = document.createElement('span');
        addrBadge.className = user.address ? 'badge bg-success badge-custom' : 'badge bg-secondary badge-custom';
        addrBadge.innerHTML = user.address ? '<i class="bi bi-check-circle"></i> True' : '<i class="bi bi-x-circle"></i> False';
        addrStatusCell.appendChild(addrBadge);

        // Действия
        const actionsCell = document.createElement('td');
        const downloadBtn = document.createElement('button');
        downloadBtn.className = 'btn btn-sm btn-outline-primary';
        downloadBtn.innerHTML = '<i class="bi bi-download"></i>';
        downloadBtn.title = 'Скачать файлы';
        downloadBtn.onclick = () => downloadUserFiles(user.user_id);
        actionsCell.appendChild(downloadBtn);

        row.appendChild(checkboxCell);
        row.appendChild(userIdCell);
        row.appendChild(citizenshipCell);
        row.appendChild(docFilesCell);
        row.appendChild(docStatusCell);
        row.appendChild(addrFilesCell);
        row.appendChild(addrStatusCell);
        row.appendChild(actionsCell);

        tbody.appendChild(row);
    });
}

// Рендеринг пагинации
function renderPagination(currentPage, totalPages) {
    const pagination = document.getElementById('pagination');
    pagination.innerHTML = '';

    // Кнопка "Предыдущая"
    const prevLi = document.createElement('li');
    prevLi.className = `page-item ${currentPage === 1 ? 'disabled' : ''}`;
    prevLi.innerHTML = `<a class="page-link" href="#" onclick="loadUsers(${currentPage - 1}); return false;">Предыдущая</a>`;
    pagination.appendChild(prevLi);

    // Генерация номеров страниц
    const maxButtons = 5;
    let startPage = Math.max(1, currentPage - Math.floor(maxButtons / 2));
    let endPage = Math.min(totalPages, startPage + maxButtons - 1);

    if (endPage - startPage + 1 < maxButtons) {
        startPage = Math.max(1, endPage - maxButtons + 1);
    }

    // Первая страница
    if (startPage > 1) {
        const li = document.createElement('li');
        li.className = 'page-item';
        li.innerHTML = `<a class="page-link" href="#" onclick="loadUsers(1); return false;">1</a>`;
        pagination.appendChild(li);

        if (startPage > 2) {
            const dots = document.createElement('li');
            dots.className = 'page-item disabled';
            dots.innerHTML = '<a class="page-link">...</a>';
            pagination.appendChild(dots);
        }
    }

    // Номера страниц
    for (let i = startPage; i <= endPage; i++) {
        const li = document.createElement('li');
        li.className = `page-item ${i === currentPage ? 'active' : ''}`;
        li.innerHTML = `<a class="page-link" href="#" onclick="loadUsers(${i}); return false;">${i}</a>`;
        pagination.appendChild(li);
    }

    // Последняя страница
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            const dots = document.createElement('li');
            dots.className = 'page-item disabled';
            dots.innerHTML = '<a class="page-link">...</a>';
            pagination.appendChild(dots);
        }

        const li = document.createElement('li');
        li.className = 'page-item';
        li.innerHTML = `<a class="page-link" href="#" onclick="loadUsers(${totalPages}); return false;">${totalPages}</a>`;
        pagination.appendChild(li);
    }

    // Кнопка "Следующая"
    const nextLi = document.createElement('li');
    nextLi.className = `page-item ${currentPage === totalPages ? 'disabled' : ''}`;
    nextLi.innerHTML = `<a class="page-link" href="#" onclick="loadUsers(${currentPage + 1}); return false;">Следующая</a>`;
    pagination.appendChild(nextLi);
}

// Переключение выбора пользователя
function toggleUserSelection(userId) {
    if (selectedUsers.has(userId)) {
        selectedUsers.delete(userId);
    } else {
        selectedUsers.add(userId);
    }
}

// Выбрать все на странице
function toggleSelectAll() {
    const selectAll = document.getElementById('select-all');
    const checkboxes = document.querySelectorAll('#users-table-body input[type="checkbox"]');

    checkboxes.forEach(checkbox => {
        checkbox.checked = selectAll.checked;
        const userId = parseInt(checkbox.value);
        if (selectAll.checked) {
            selectedUsers.add(userId);
        } else {
            selectedUsers.delete(userId);
        }
    });
}

// Скачать файлы пользователя
async function downloadUserFiles(userId) {
    // Находим кнопку и показываем индикатор загрузки
    const button = event.target.closest('button');
    const originalContent = button.innerHTML;
    button.disabled = true;
    button.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Скачивание...';

    try {
        const response = await fetch(`/api/download/user?user_id=${userId}`, {
            method: 'POST'
        });

        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || 'Ошибка скачивания файлов');
        }

        const data = await response.json();

        if (data.success) {
            let message = `✓ Файлы пользователя ${userId} успешно скачаны!\n\n`;
            message += `Путь: ${data.path}\n`;
            message += `Скачано файлов: ${data.files_downloaded}\n`;
            if (data.document_success) {
                message += `✓ Document файлы скачаны\n`;
            }
            if (data.address_success) {
                message += `✓ Address файлы скачаны\n`;
            }
            alert(message);

            // Перезагружаем таблицу для обновления статусов
            loadUsers(currentPage);
        } else {
            throw new Error(data.error || 'Не удалось скачать файлы');
        }
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка: ' + error.message);
    } finally {
        // Восстанавливаем кнопку
        button.disabled = false;
        button.innerHTML = originalContent;
    }
}

// Скачать все выбранные
async function downloadAll() {
    if (selectedUsers.size === 0) {
        alert('Выберите хотя бы одного пользователя');
        return;
    }

    const users = Array.from(selectedUsers);
    let message = `Файлы ${users.length} пользователей находятся в:\n\n`;

    for (const userId of users) {
        try {
            const response = await fetch(`/api/download?user_id=${userId}`);
            if (response.ok) {
                const data = await response.json();
                message += `User ${userId}: ${data.path}\n`;
            }
        } catch (error) {
            console.error(`Ошибка для пользователя ${userId}:`, error);
        }
    }

    alert(message);
}

// === Управление скачиванием ===

let progressInterval = null;

// Запустить скачивание
async function startDownload() {
    try {
        const response = await fetch('/api/download/start', {
            method: 'POST'
        });

        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || 'Ошибка запуска скачивания');
        }

        // Показываем панель прогресса
        document.getElementById('download-progress-container').style.display = 'block';

        // Отключаем кнопку запуска и включаем кнопку остановки
        document.getElementById('start-download-btn').disabled = true;
        document.getElementById('stop-download-btn').disabled = false;

        // Запускаем обновление прогресса
        if (progressInterval) {
            clearInterval(progressInterval);
        }
        progressInterval = setInterval(updateProgress, 1000);

        alert('Скачивание запущено!');
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка: ' + error.message);
    }
}

// Остановить скачивание
async function stopDownload() {
    try {
        const response = await fetch('/api/download/stop', {
            method: 'POST'
        });

        if (!response.ok) {
            throw new Error('Ошибка остановки скачивания');
        }

        // Останавливаем обновление прогресса
        if (progressInterval) {
            clearInterval(progressInterval);
            progressInterval = null;
        }

        // Включаем кнопку запуска и отключаем кнопку остановки
        document.getElementById('start-download-btn').disabled = false;
        document.getElementById('stop-download-btn').disabled = true;

        alert('Скачивание остановлено!');
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка: ' + error.message);
    }
}

// Обновить прогресс
async function updateProgress() {
    try {
        const response = await fetch('/api/download/progress');
        if (!response.ok) {
            throw new Error('Ошибка получения прогресса');
        }

        const data = await response.json();

        // Обновляем статус
        const statusBadge = document.getElementById('download-status');
        statusBadge.textContent = data.status;
        statusBadge.className = 'badge ' + getStatusClass(data.status);

        // Обновляем прогресс-бар
        const progressPercent = data.progress_percent || 0;
        const progressBar = document.getElementById('download-progress-bar');
        progressBar.style.width = progressPercent + '%';
        progressBar.textContent = progressPercent.toFixed(1) + '%';

        // Обновляем статистику
        document.getElementById('progress-processed').textContent = data.processed_users;
        document.getElementById('progress-total').textContent = data.total_users;
        document.getElementById('progress-successful').textContent = data.successful_users;
        document.getElementById('progress-files').textContent = data.successful_files;
        document.getElementById('progress-duration').textContent = formatDuration(data.duration_seconds);

        // Если скачивание завершено или остановлено
        if (data.status === 'completed' || data.status === 'idle' || data.status === 'failed') {
            if (progressInterval) {
                clearInterval(progressInterval);
                progressInterval = null;
            }

            document.getElementById('start-download-btn').disabled = false;
            document.getElementById('stop-download-btn').disabled = true;

            // Перезагружаем список пользователей
            if (data.status === 'completed') {
                setTimeout(() => {
                    loadUsers(currentPage);
                    alert('Скачивание завершено!');
                }, 1000);
            }
        }
    } catch (error) {
        console.error('Ошибка обновления прогресса:', error);
    }
}

// Получить CSS класс для статуса
function getStatusClass(status) {
    switch (status) {
        case 'running':
            return 'bg-primary';
        case 'completed':
            return 'bg-success';
        case 'failed':
            return 'bg-danger';
        case 'paused':
            return 'bg-warning';
        default:
            return 'bg-secondary';
    }
}

// Форматировать длительность
function formatDuration(seconds) {
    if (seconds < 60) {
        return Math.round(seconds) + 's';
    } else if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60);
        const secs = Math.round(seconds % 60);
        return minutes + 'm ' + secs + 's';
    } else {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        return hours + 'h ' + minutes + 'm';
    }
}

// Проверяем статус при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Проверяем текущий статус скачивания
    fetch('/api/download/progress')
        .then(response => response.json())
        .then(data => {
            if (data.status === 'running') {
                document.getElementById('download-progress-container').style.display = 'block';
                document.getElementById('start-download-btn').disabled = true;
                document.getElementById('stop-download-btn').disabled = false;

                // Запускаем обновление прогресса
                progressInterval = setInterval(updateProgress, 1000);
            }
        })
        .catch(error => console.error('Ошибка проверки статуса:', error));
});
