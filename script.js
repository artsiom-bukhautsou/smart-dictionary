const storedUsername = localStorage.getItem("username");
const storedPassword = localStorage.getItem("password");
const deckId = localStorage.getItem("deckId");

if (!storedUsername || !storedPassword) {
    // Credentials are not stored, prompt the user to enter them
    const username = prompt("Enter your username:");
    const password = prompt("Enter your password:");
    const deckId = prompt("Enter your deck name:");

    // Store credentials in localStorage
    localStorage.setItem("username", username);
    localStorage.setItem("password", password);
    localStorage.setItem("deckId", deckId);
}

async function translateWord() {
    const wordInput = document.getElementById("wordInput").value;

    const translationContainer = document.getElementById("translationContainer");
    const loader = document.getElementById("loader");
    loader.classList.remove("loader-hidden");
    const translateInput = document.getElementById("translate-input");
    translateInput.classList.add("loader-working");
    translationContainer.classList.add("loader-working");
    const widget1 = document.getElementById("widget-1");
    widget1.classList.add("loader-working");
    widget.pause()

    try {
        const response = await fetch("http://translator.artem.codes:8080/translations", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Authorization": "Basic " + btoa(`${storedUsername}:${storedPassword}`),
                "Deck-Id": deckId
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
    } finally {
        loader.classList.add("loader-hidden");
        translateInput.classList.remove("loader-working");
        translationContainer.classList.remove("loader-working");
        widget1.classList.remove("loader-working");
    }
    onYouglishAPIReady(wordInput)
}

function displayTranslation(translation) {
    const translationContainer = document.getElementById("translationContainer");

    const html = `
            <p><b>Meaning</b>: ${translation.meaning}</p>
            <p><b>Examples</b>: ${translation.examples.join(", ")}</p>
            <p><b>Russian</b> Translation: ${translation.russianTranslation}</p>
            <p><b>Meaning</b> in Russian: ${translation.meaningRussian}</p>
            <p><b>Examples</b> in Russian: ${translation.examplesRussian.join(", ")}</p>
    `;

    translationContainer.innerHTML = html;
}

// 2. This code loads the widget API code asynchronously.
var tag = document.createElement('script');

tag.src = "https://youglish.com/public/emb/widget.js";
var firstScriptTag = document.getElementsByTagName('script')[0];
firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

// 3. This function creates a widget after the API code downloads.
var widget;

function onYouglishAPIReady(wordInput) {
    widget = new YG.Widget("widget-1", {
        width: 640,
        components: 92, //search box & caption
        events: {
            'onFetchDone': onFetchDone,
            'onVideoChange': onVideoChange,
            'onCaptionConsumed': onCaptionConsumed
        }
    });
    // 4. process the query
    widget.fetch(wordInput);
}


var views = 0, curTrack = 0, totalTracks = 0;

// 5. The API will call this method when the search is done
function onFetchDone(event) {
    if (event.totalResult === 0) alert("No result found");
    else totalTracks = event.totalResult;
}

// 6. The API will call this method when switching to a new video.
function onVideoChange(event) {
    curTrack = event.trackNumber;
    views = 0;
}

// 7. The API will call this method when a caption is consumed.
function onCaptionConsumed(event) {
    // if (++views < 3)
    //     widget.replay();
    // else
    if (curTrack < totalTracks)
        widget.next();
}