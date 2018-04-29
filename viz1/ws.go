// ws.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Foo struct {
	Id   int
	Name string
	Bar  bool
}

func ticker(tick chan int) {
	for {
		tick <- 1
		time.Sleep(750 * time.Millisecond)
	}
}
func main() {

	var foo *Foo
	tick := make(chan int)

	go ticker(tick)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity
		idx := 0
		for {
			foo = &Foo{
				Id:   idx,
				Name: "Test Message",
				Bar:  true,
			}

			j, _ := json.Marshal(foo)

			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(j))

			// Write message back to browser
			if err := conn.WriteMessage(websocket.TextMessage, j); err != nil {
				return
			}
			idx++

			<-tick
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ws.html")
	})

	http.ListenAndServe(":3000", nil)
}
