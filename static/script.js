
let SOCKET = null;


const BLOCKS = [" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"];


function putError(errMsg) {
    let header = document.getElementById("header");
    header.className = "error";
    header.innerHTML = errMsg;
}


function sockOpen() {
    console.log("Websocket connection was opened.");
}

function sockClose() {
    putError("Websocket connection was closed.");
}

function sockError(error) {
    console.log("Socket Error: ", error);
}

function sockMessage(msg) {
    drawSoundCards(JSON.parse(msg.data));
}


function intToB36(i) {
    i = i % 36;
    if (i <= 9) {
        return String(i);
    }

    let offset = "A".charCodeAt(0);
    return String.fromCharCode(offset + (i - 10));
}


function getEffectsString(sound) {
    let str = "";
    let wrapName = (n) => `<span class="effect-name">${n}</span>`;
    let wrapVal = (v) => `<span class="effect-value">${v}</span>`;

    // number of chops
    str += wrapName("C")
        + wrapVal(intToB36(sound["chops"]));

    // speed ratio
    str += wrapName("S")
        + wrapVal(intToB36(sound["speedN"]) + intToB36(sound["speedD"]));

    // mince
    str += wrapName("M")
        + wrapVal(intToB36(sound["minceSize"]) + intToB36(sound["minceInterval"]));

    // pan
    str += wrapName("P")
        + wrapVal(intToB36(sound["pan"]));

    // volume
    str += wrapName("V")
        + wrapVal(intToB36(sound["volume"]));

    return str;
}


function drawSoundCards(sounds) {
    // first generate list of DOM elements
    // let cardColumn = document.getElementById("center-column");
    let cardColumn = document.getElementById("sound-card-wrapper");
    cardColumn.innerHTML = "";

    let i = 0;
    for (sound of sounds) {
        let card = document.createElement("div");
        card.className = "sound-card";

        let cardLeft = document.createElement("div");
        cardLeft.className = "sound-card-left";
        card.appendChild(cardLeft);
        let cardRight = document.createElement("div");
        cardRight.className = "sound-card-right";
        card.appendChild(cardRight);

        let index = document.createElement("div");
        index.className = "sound-card-index";
        index.innerHTML = i;
        cardLeft.appendChild(index);

        let path = document.createElement("div");
        path.className = "sound-card-path";
        path.innerHTML = sound["path"].replace(/^.*[\\\/]/, '');
        cardRight.appendChild(path);

        let effects = document.createElement("div");
        effects.className = "sound-card-effects";
        effects.innerHTML = getEffectsString(sound);
        cardRight.appendChild(effects);

        cardColumn.appendChild(card);
        i++;
    }
}

function pickNewSoundFile() {
    let elem = document.getElementById("new-sound-picker");
    let evt = document.createEvent("MouseEvents");
    evt.initEvent("click", true, false);
    elem.dispatchEvent(evt);
}

function loadNewSound(path) {
    console.log(path);
}

window.onload = () => {
    // initialize websocket
    if (typeof WEBSERVER_PORT === "undefined") {
        putError("WEBSERVER_PORT was not defined, cannot open websocket for communication with server.");
        return;
    }

    socket = new WebSocket(`ws://localhost:${WEBSERVER_PORT}/socket`);
    socket.onopen = sockOpen;
    socket.onclose = sockClose;
    socket.onerror = sockError;
    socket.onmessage = sockMessage;

    // add new sound input event listener
    const soundInput = document.getElementById("new-sound-picker");
    soundInput.addEventListener("input", loadNewSound);
}