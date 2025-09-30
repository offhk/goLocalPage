package main

import (
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	selectedOption = "Option 1" // Default selected option
	mu             sync.Mutex
	clients        = make(map[*websocket.Conn]bool) // Connected clients
	upgrader       = websocket.Upgrader{}           // WebSocket upgrader
)

func main() {
	http.HandleFunc("/", editPage)
	http.HandleFunc("/ws", handleWebSocket)

	// Start the server on port 8080
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func editPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Local Webpage Editor</title>
        <script>
            const socket = new WebSocket("ws://" + window.location.host + "/ws");
            socket.onmessage = function(event) {
                document.getElementById('currentContent').innerText = event.data;
            };

            function updateSelection() {
                const selectElement = document.getElementById('optionSelect');
                const selectedValue = selectElement.value;
                socket.send(selectedValue);
            }

            // Refresh the page every 5 seconds
            setInterval(function() {
                location.reload();
            }, 10000);
        </script>
    </head>
    <body>
        <h1>Select an Option</h1>
        <h3>Select an Option:</h3>
        <select id="optionSelect" onchange="updateSelection()">
            <option value="Option 1" {{if eq .SelectedOption "Option 1"}}selected{{end}}>Option 1</option>
            <option value="Option 2" {{if eq .SelectedOption "Option 2"}}selected{{end}}>Option 2</option>
            <option value="Option 3" {{if eq .SelectedOption "Option 3"}}selected{{end}}>Option 3</option>
        </select><br>
        <h2>Current Selection:</h2>
        <p id="currentContent">{{.SelectedOption}}</p>
    </body>
    </html>
    `
	mu.Lock()
	defer mu.Unlock()

	data := struct {
		SelectedOption string
	}{
		SelectedOption: selectedOption,
	}

	t := template.Must(template.New("edit").Parse(tmpl))
	t.Execute(w, data)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	log.Println("New client connected")

	// Send the current selected option to the newly connected client
	mu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, []byte(selectedOption))
	mu.Unlock()
	if err != nil {
		log.Println("Error sending initial message:", err)
		return
	}

	for {
		// Read message from the WebSocket
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			delete(clients, conn)
			break
		}

		mu.Lock()
		selectedOption = string(msg) // Convert message to string
		mu.Unlock()

		log.Printf("Broadcasting message: %s\n", selectedOption)

		// Broadcast the message to all clients
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Println("Error broadcasting message:", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
