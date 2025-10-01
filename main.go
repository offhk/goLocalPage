package main

import (
	"encoding/json"
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
	log.Println("Server started on :9000")
	err := http.ListenAndServe(":9000", nil)
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


<style>

body {
background-color: black;
}

label {
  display: inline-block;
  width: 20px;
  text-align: right;
  color: aqua;
  font: 50px;
  margin-right: 5px;
}

select {
  display: inline-block;
  vertical-align: middle;
  font-weight: bold;
}

</style>



    <script>
        const socket = new WebSocket("ws://" + window.location.host + "/ws");
socket.onmessage = function(event) {
    const data = JSON.parse(event.data);

    // Check if data is an object with number keys (full state)
    if (typeof data === "object" && !Array.isArray(data)) {
        // Update all selects based on keysgit
        for (const key in data) {
            const element = document.getElementById('optionSelect' + key);
            if (element) {
                element.value = data[key];
            }
        }
        return;
    }

    // Otherwise, expect { index, value } object for one selection update
    if (data.index === undefined || data.value === undefined) {
        console.warn('Received message missing index or value:', data);
        return;
    }

    const element = document.getElementById('optionSelect' + data.index);
    if (element) {
        element.value = data.value;
    } else {
        console.warn('No element found for id:', 'optionSelect' + data.index);
    }
};

        function updateSelection(index) {
            const selectElement = document.getElementById('optionSelect' + index);
            const selectedValue = selectElement.value;
            socket.send(JSON.stringify({ index: index, value: selectedValue }));
        }

        // Refresh the page every 10 seconds
        setInterval(function() {
            location.reload();
        }, 7000); // Changed from 5000 to 10000 milliseconds
    </script>
</head>

<body>
    {{range $i := seq 1 14}}
        <label for="optionSelect{{$i}}">{{$i}}</label>
        <select id="optionSelect{{$i}}" onchange="updateSelection({{$i}})">
            <option value="Option 1"></option>
            <option value="Option 2">YC</option>
            <option value="Option 3">Sern</option>
            <option value="Option 4">Wtm</option>
            <option value="Option 5">San</option>
            <option value="Option 6">Ln</option>
            <option value="Option 7">Bt</option>
            <option value="Option 8">WW</option>
            <option value="Option 9">Kv</option>
            <option value="Option 10">Kg</option>
            <option value="Option 11">??</option>
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

var selections = make(map[int]string)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	mu.Lock()
	// Send entire current selections map as JSON on connect
	allSelectionsJSON, _ := json.Marshal(selections)
	mu.Unlock()
	conn.WriteMessage(websocket.TextMessage, allSelectionsJSON)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read null:", err)
			delete(clients, conn)
			break
		}

		var update struct {
			Index int
			Value string
		}
		err = json.Unmarshal(msg, &update)
		if err != nil {
			log.Println("JSON unmarshal error:", err)
			continue
		}

		mu.Lock()
		selections[update.Index] = update.Value
		mu.Unlock()

		// Broadcast update to all clients
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}
