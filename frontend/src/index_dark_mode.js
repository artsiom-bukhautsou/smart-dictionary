document.addEventListener('DOMContentLoaded', function() {
    const themeToggle = document.getElementById('theme-toggle');
    const htmlRoot = document.getElementById('html-root');

    // Check if dark mode is already enabled in localStorage
    if (localStorage.getItem('dark-mode') === 'enabled') {
        document.body.classList.add('dark');
        themeToggle.textContent = '☀️';
    } else {
        themeToggle.textContent = '🌙';
    }

    themeToggle.addEventListener('click', function() {
        document.body.classList.toggle('dark');

        // Save the user's preference in localStorage
        if (document.body.classList.contains('dark')) {
            localStorage.setItem('dark-mode', 'enabled');
            themeToggle.textContent = '☀️';
        } else {
            localStorage.setItem('dark-mode', 'disabled');
            themeToggle.textContent = '🌙';
        }
    });
});

