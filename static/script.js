
let SOCKET = null;


function putError(errMsg) {
    let header = document.getElementById("header");
    header.className = "error";
    header.innerHTML = errMsg;
}


function sockOpen() {
    // TODO request current state
    console.log("Websocket connection was opened.");
}

function sockClose() {
    putError("Websocket connection was closed.");
}

function sockError(error) {
    console.log("Socket Error: ", error);
}

function sockMessage(msg) {
    // TODO handle message
    console.log(msg);
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

    socket.send("test");
}