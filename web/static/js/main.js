document.addEventListener('DOMContentLoaded', function () {
    setupLogoutConfirm();
    setupRelativeTimestamps();
    setupCharCounter();
});

function setupLogoutConfirm() {
    var form = document.querySelector('.logout-form');
    var modal = document.getElementById('logout-modal');
    if (!form || !modal) return;

    var confirmButton = document.getElementById('logout-modal-confirm');
    var closeTriggers = modal.querySelectorAll('[data-modal-close]');

    function closeModal() {
        modal.hidden = true;
    }

    form.addEventListener('submit', function (event) {
        event.preventDefault();
        modal.hidden = false;
        confirmButton.focus();
    });

    confirmButton.addEventListener('click', function () {
        closeModal();
        form.submit();
    });

    closeTriggers.forEach(function (el) {
        el.addEventListener('click', closeModal);
    });

    document.addEventListener('keydown', function (event) {
        if (event.key === 'Escape' && !modal.hidden) {
            closeModal();
        }
    });
}

function setupRelativeTimestamps() {
    var times = document.querySelectorAll('time[datetime]');

    times.forEach(function (el) {
        var date = new Date(el.getAttribute('datetime'));
        if (isNaN(date.getTime())) return;
        el.textContent = formatRelativeTime(date);
    });
}

function formatRelativeTime(date) {
    var seconds = Math.floor((Date.now() - date.getTime()) / 1000);
    if (seconds < 60) return 'just now';

    var units = [
        ['year', 31536000],
        ['month', 2592000],
        ['day', 86400],
        ['hour', 3600],
        ['minute', 60],
    ];

    for (var i = 0; i < units.length; i++) {
        var name = units[i][0];
        var secondsInUnit = units[i][1];
        var count = Math.floor(seconds / secondsInUnit);
        if (count >= 1) {
            return count + ' ' + name + (count > 1 ? 's' : '') + ' ago';
        }
    }

    return 'just now';
}

function setupCharCounter() {
    var textarea = document.querySelector('.post-form__body');
    var counter = document.querySelector('.post-form__char-count');
    if (!textarea || !counter) return;

    var update = function () {
        counter.textContent = textarea.value.length + ' characters';
    };

    textarea.addEventListener('input', update);
    update();
}
