const storedUsername = localStorage.getItem("username"); const storedPassword = localStorage.getItem("password"); if (!storedUsername || !storedPassword) {
    // Credentials are not stored, prompt the user to enter them
    const username = prompt("Enter your username:");
    const password = prompt("Enter your password:");

    // Store credentials in localStorage
    localStorage.setItem("username", username);
    localStorage.setItem("password", password);
}

async function translateWord() {
    const wordInput = document.getElementById("wordInput").value;

    const translationContainer = document.getElementById("translationContainer");
    translationContainer.innerHTML = "<p>Loading...</p>";

    try {
        const response = await fetch("http://51.83.128.169/:8080/translations", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Authorization": "Basic " + btoa(`${storedUsername}:${storedPassword}`),
            },
            body: JSON.stringify({word: wordInput}),
        });

        const responseData = await response.json();

        if (response.ok) {
            displayTranslation(responseData);
        } else {
            translationContainer.innerHTML = `<p>Error: ${responseData.message}</p>`;
        }
    } catch (error) {
        translationContainer.innerHTML = `<p>Error: ${error.message}</p>`;
    }
}

function displayTranslation(translation) {
    const translationContainer = document.getElementById("translationContainer");

    const html = `
        <p>Meaning: ${translation.meaning}</p>
        <p>Examples: ${translation.examples.join(", ")}</p>
        <p>Russian Translation: ${translation.russianTranslation}</p>
        <p>Meaning in Russian: ${translation.meaningRussian}</p>
        <p>Examples in Russian: ${translation.examplesRussian.join(", ")}</p>
    `;

    translationContainer.innerHTML = html;
}