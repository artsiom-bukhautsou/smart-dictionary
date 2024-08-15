document.addEventListener('DOMContentLoaded', function () {
    // Check if user is logged in
    if (!localStorage.getItem('username') || !localStorage.getItem('password')) {
        // To prevent infinite redirection loop, use sessionStorage to track the redirection status
        if (!sessionStorage.getItem('redirected')) {
            sessionStorage.setItem('redirected', 'true');
            window.location.href = 'index.html'; // Redirect to login if not logged in
        }
    } else {
        // Clear the redirection flag if user is logged in
        sessionStorage.removeItem('redirected');
    }
});

function logout() {
    localStorage.removeItem('username');
    localStorage.removeItem('password');
    window.location.href = 'index.html'; // Redirect to login page
}
