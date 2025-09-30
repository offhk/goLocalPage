package main

import (
	"html/template"
	"net/http"
	"sync"
)

var (
	content = "Edit this text."
	mu      sync.Mutex
)

func main() {
	http.HandleFunc("/", editPage)
	http.HandleFunc("/save", saveContent)
	http.HandleFunc("/getContent", getContent)

	// Start the server on port 8080
	http.ListenAndServe(":8080", nil)
}

func editPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Local Webpage Editor</title>
        <script>
            function fetchContent() {
                fetch('/getContent')
                    .then(response => response.text())
                    .then(data => {
                        document.getElementById('currentContent').innerText = data;
                    });
            }
            setInterval(fetchContent, 5000); // Fetch content every 5 seconds
        </script>
    </head>
    <body>
        <h1>Edit the Content 002</h1>
        <form action="/save" method="post">
            <textarea name="content" rows="10" cols="50">{{.}}</textarea><br>
            <input type="submit" value="Save">
        </form>
        <h2>Current Content:</h2>
        <p id="currentContent">{{.}}</p>
    </body>
    </html>
    `
	mu.Lock()
	defer mu.Unlock()

	t := template.Must(template.New("edit").Parse(tmpl))
	t.Execute(w, content)
}

func saveContent(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if r.Method == http.MethodPost {
		r.ParseForm()
		content = r.FormValue("content")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getContent(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Write([]byte(content))
}
