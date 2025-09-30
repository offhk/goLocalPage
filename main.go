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
	// Create a new template and register the seq function
	tmpl := template.Must(template.New("edit").Funcs(template.FuncMap{
		"seq": seq, // Register the seq function
	}).Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Local Webpage Editor</title>
    <script>
        const socket = new WebSocket("ws://" + window.location.host + "/ws");
        socket.onmessage = function(event) {
            // Update the selection in the dropdown
            const data = JSON.parse(event.data);
            document.getElementById('optionSelect' + data.index).value = data.value;
        };

        function updateSelection(index) {
            const selectElement = document.getElementById('optionSelect' + index);
            const selectedValue = selectElement.value;
            socket.send(JSON.stringify({ index: index, value: selectedValue }));
        }

        // Refresh the page every 10 seconds
        setInterval(function() {
            location.reload();
        }, 10000); // Changed from 5000 to 10000 milliseconds
    </script>
</head>
<body>
    <h1>Select Options</h1>
    {{range $i := seq 1 14}}
        <label for="optionSelect{{$i}}">{{$i}}</label>
        <select id="optionSelect{{$i}}" onchange="updateSelection({{$i}})">
            <option value="Option 1">Option 1</option>
            <option value="Option 2">Option 2</option>
            <option value="Option 3">Option 3</option>
        </select><br>
    {{end}}
</body>
</html>
`))

	mu.Lock()
	defer mu.Unlock()

	data := struct {
		SelectedOption string
	}{
		SelectedOption: selectedOption,
	}

	tmpl.Execute(w, data)
}

// Helper function to generate a sequence of numbers
func seq(start, end int) []int {
	s := make([]int, end-start+1)
	for i := start; i <= end; i++ {
		s[i-start] = i
	}
	return s
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
