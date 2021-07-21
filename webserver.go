package main

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
)

type TemplateData struct {
	Port int
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var templates *template.Template

var templateData TemplateData

func indexEndpoint(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", templateData)
}

func handleWebsocketConn(conn *websocket.Conn) {
	for {
		msgType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		// print any incoming messages
		fmt.Println(string(p))

		// echo message back to client
		if err := conn.WriteMessage(msgType, p); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func websocketEndpoint(w http.ResponseWriter, r *http.Request) {
	// for now accept any type of inconming connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}

	handleWebsocketConn(wsConn)
}

func runWebServer(port int) {
	templateData = TemplateData{Port: port}

	// initialze HTML templates folder
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// initialize static file server
	fs := http.FileServer(http.Dir("./static/"))

	// initialize endpoint handlers
	http.HandleFunc("/", indexEndpoint)
	http.HandleFunc("/socket", websocketEndpoint)
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
