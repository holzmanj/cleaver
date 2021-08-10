package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
)

type TemplateData struct {
	Port int
}

type ConnList struct {
	conns []*websocket.Conn
	mut   sync.Mutex
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var templates *template.Template

var templateData TemplateData

var activeConnections ConnList

func dumpChainConfigs() []byte {
	var configs []ChainConfig
	for _, chain := range Chains {
		configs = append(configs, chain.Config)
	}

	bytes, err := json.Marshal(configs)
	if err != nil {
		fmt.Println(err)
		return []byte(`{"error": "failed to dump chain configs to json"}`)
	}
	return bytes
}

func PushChainConfigs() {
	activeConnections.mut.Lock()
	for _, conn := range activeConnections.conns {
		conn.WriteMessage(websocket.TextMessage, dumpChainConfigs())
	}
	activeConnections.mut.Unlock()
}

func addActiveConnection(conn *websocket.Conn) {
	activeConnections.mut.Lock()
	activeConnections.conns = append(activeConnections.conns, conn)
	activeConnections.mut.Unlock()
}

func removeActiveConnection(conn *websocket.Conn) {
	activeConnections.mut.Lock()
	defer activeConnections.mut.Unlock()

	// find index of conn in active connections
	i := -1
	for ci := range activeConnections.conns {
		if conn == activeConnections.conns[ci] {
			i = ci
			break
		}
	}

	// conn wasn't found in active connections list
	if i == -1 {
		return
	}

	// remove item
	activeConnections.conns[i] = activeConnections.conns[len(activeConnections.conns)-1]
	activeConnections.conns = activeConnections.conns[:len(activeConnections.conns)-1]
}

func handleSocketConn(conn *websocket.Conn) {
	addActiveConnection(conn)
	defer removeActiveConnection(conn)

	conn.WriteMessage(websocket.TextMessage, dumpChainConfigs())
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

func indexEndpoint(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", templateData)
}

func socketEndpoint(w http.ResponseWriter, r *http.Request) {
	// for now accept any type of inconming connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}

	handleSocketConn(wsConn)
}

func runWebServer(port int) {
	templateData = TemplateData{Port: port}

	// initialze HTML templates folder
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// initialize static file server
	fs := http.FileServer(http.Dir("./static/"))

	// initialize endpoint handlers
	http.HandleFunc("/", indexEndpoint)
	http.HandleFunc("/socket", socketEndpoint)
	http.Handle("/static/", http.StripPrefix("/static", fs))

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
