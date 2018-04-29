package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// limited controls the number of tickets for debugging and testing.
func limited(done chan int, tick chan int) {
	for i := 0; i < 720; i++ {
		<-tick
	}
	done <- 1
}

func ticker(tick chan int) {
	// previous := time.Now()
	for {
		tick <- 1
		// current := time.Now()
		// elapsed := current.Sub(previous)
		// previous = current

		// log.Printf("step %f", elapsed)
		time.Sleep(301 * time.Millisecond)
	}
}

func handleInput(done chan int) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("-> ")
		char, _, err := reader.ReadRune()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(char)

		switch char {
		case 'h':
			fmt.Println("h: help, q: quit, f: faster, s: slower")
			break
		case 'q':
			fmt.Println("goodbye!")
			done <- 1
			break
		case 'f':
			fmt.Println("faster....")
			break
		case 's':
			fmt.Println(".... slower")
			break
		}
	}
}

func handleRuntime(t1 Track, tick chan int, render chan Object) {
	for {
		fmt.Println("runtime...")
		t1.Tick()

		t1.Render(render)
		<-tick
	}
}

func handleRender(hub *Hub, tick chan int, render chan Object) {
	// previous := time.Now()
	var v Object
	for {
		v = <-render

		hub.broadcastAll(v)
		fmt.Println(v)
	}
}

func handleServer(hub *Hub) {
	assets := http.StripPrefix("/", http.FileServer(http.Dir("client/")))
	http.Handle("/", assets)
	http.HandleFunc("/ws", hub.handleWebSocket)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	v1 := NewVehicle("AAA", "Model X", "drive", 15.0)
	v2 := NewVehicle("BBB", "Model X", "drive", 35.0)
	v3 := NewVehicle("CCC", "Model S", "drive", 25.0)
	v4 := NewVehicle("ZZZ", "Leaf", "drive", 70.0)

	c1 := NewCharger("A", "t1", "online")
	c2 := NewCharger("B", "t1", "online")
	c3 := NewCharger("C", "t2", "online")

	t1 := NewCircularTrack("T", Points{180.0, 135.0}, 120.0)
	t1.Add(v1)
	t1.Add(v2)
	t1.Add(v3)
	t1.Add(v4)
	t1.Add(c1)
	t1.Add(c2)
	t1.Add(c3)

	t1.RandomizeObjects()

	tick := make(chan int)
	done := make(chan int)

	render := make(chan Object)

	go ticker(tick)
	// go limited(done, tick)
	go handleInput(done)
	go handleRuntime(t1, tick, render)

	hub := newHub()
	go hub.run()
	go handleRender(hub, tick, render)
	go handleServer(hub)

	<-done
}
