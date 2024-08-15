document.addEventListener('DOMContentLoaded', function () {
    // Check if user is already logged in
    if (localStorage.getItem('username')) {
        window.location.href = 'translate.html'; // Redirect to Hello World page if logged in
    }
});

const signInForm = document.getElementById('sign-in-form');
const signUpForm = document.getElementById('sign-up-form');
const formTitle = document.getElementById('form-title');
const toggleFormText = document.getElementById('toggle-form-text');
const toggleFormLink = document.getElementById('toggle-form-link');

toggleFormLink.addEventListener('click', (e) => {
    e.preventDefault();
    if (signInForm.classList.contains('hidden')) {
        signInForm.classList.remove('hidden');
        signUpForm.classList.add('hidden');
        formTitle.textContent = 'Sign in to your account';
        toggleFormText.textContent = 'Not a member?';
        toggleFormLink.textContent = 'Sign up here';
    } else {
        signInForm.classList.add('hidden');
        signUpForm.classList.remove('hidden');
        formTitle.textContent = 'Sign up for an account';
        toggleFormText.textContent = 'Already have an account?';
        toggleFormLink.textContent = 'Sign in here';
    }
});

async function handleSignIn(event) {
    event.preventDefault();
    const email = document.getElementById('sign-in-email').value;
    const password = document.getElementById('sign-in-password').value;
    console.log(`Sign In - Email: ${email}, Password: ${password}`);

    const payload = {
        username: email,
        password: password
    };

    try {
        const response = await fetch("http://localhost:8080/auth/signin", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            // Store in localStorage
            localStorage.setItem("username", email);
            localStorage.setItem("password", password); // Ideally store a token, not a password
            window.location.href = 'translate.html';
        } else {
            alert("Invalid username or password");
        }
    } catch (error) {
        alert("Error during sign-in: " + error.message);
    }
}

async function handleSignUp(event) {
    event.preventDefault();
    const email = document.getElementById('sign-up-email').value;
    const password = document.getElementById('sign-up-password').value;
    console.log(`Sign Up - Email: ${email}, Password: ${password}`);

    const payload = {
        username: email,
        password: password
    };

    try {
        const response = await fetch("http://localhost:8080/auth/signup", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            localStorage.setItem("username", email);
            localStorage.setItem("password", password); // Ideally store a token, not a password
            window.location.href = 'translate.html';
        } else {
            alert("Sign-up failed, please try again");
        }
    } catch (error) {
        alert("Error during sign-up: " + error.message);
    }
}
