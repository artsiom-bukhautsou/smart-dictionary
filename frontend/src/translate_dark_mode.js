const themeToggleBtn = document.getElementById('themeToggle');
const body = document.body;

// Check if dark mode is already enabled in localStorage
if (localStorage.getItem('dark-mode') === 'enabled') {
    document.body.classList.add('dark');
    themeToggleBtn.textContent = '☀️';
} else {
    themeToggleBtn.textContent = '🌙';
}

function toggleTheme() {
    body.classList.toggle('dark');
    if (body.classList.contains('dark')) {
        themeToggleBtn.textContent = '☀️';
        localStorage.setItem('dark-mode', 'enabled')
    } else {
        themeToggleBtn.textContent = '🌙';
        localStorage.setItem('dark-mode', 'disabled')
    }
}