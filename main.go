package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"
)

type CircleTrack struct {
	j, k   float64
	Radius float64
}

type Vehicle struct {
	X, Y, C            float64
	Type, Rego, Status string
	Point              float64
	Speed              float64
	Direction          float64
}

type Charger struct {
	X, Y               float64
	Type, Rego, Status string
	Point              float64
}

func limited(done chan int, tick chan int) {
	for i := 0; i < 360; i++ {
		<-tick
	}
	done <- 1
}

func render(v_pipeline chan *Vehicle, c_pipeline chan *Charger) {
	// wait for new things to render on pipeline
	for {
		var v *Vehicle
		var c *Charger

		select {
		case v = <-v_pipeline:
			j, err := json.Marshal(v)
			if err != nil {
				log.Printf("got error")
			}

			log.Printf("render vehicle: %s", string(j))
		case c = <-c_pipeline:
			j, err := json.Marshal(c)
			if err != nil {
				log.Printf("got error")
			}

			log.Printf("render charger: %s", string(j))
		default:
			time.Sleep(250 * time.Millisecond)
		}

	}
}

func updateState(done chan int, tick chan int, v_pipeline chan *Vehicle, c_pipeline chan *Charger) {

	// initialize objects
	track := CircleTrack{0.0, 0.0, 5.0}

	var vehicles = []Vehicle{
		{0, 0, 90.0, "tesla x", "AAA", "drive", 2 * math.Pi, 1, -1},
		{0, 0, 90.0, "tesla x", "BBB", "drive", 2 * math.Pi, 0.75, 1},
	}
	var chargers = []Charger{
		{0, 0, "ch1", "A", "online", math.Pi / 3},
		{0, 0, "ch1", "B", "online", 3 * math.Pi / 2},
		{0, 0, "ch2", "C", "online", math.Pi},
	}

	// charger objects
	var charger *Charger
	for i := 0; i < len(chargers); i++ {
		charger = &chargers[i]
		updateCharger(track, charger)
		// fmt.Printf("updated charger idx %d, rego: %s point %f, wiht x %f y %f\n", idx, r.Rego, r.Point, r.X, r.Y)
		c_pipeline <- charger
	}

	// dynamic objects
	for {
		var vehicle *Vehicle
		// for each item to compute the state for...
		for i := 0; i < len(vehicles); i++ {
			vehicle = &vehicles[i]

			updateCharge(vehicle)
			updateDirection(vehicle, chargers)
			updateVehicle(track, vehicle)
			fmt.Printf("updated vehicle rego: %s point %f, wiht x %f y %f\n", vehicle.Rego, vehicle.Point, vehicle.X, vehicle.Y)
			v_pipeline <- vehicle
		}
		//  wait for synchronisation
		<-tick
	}
}

func inputMapper(done chan int) {

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

func ticker(tick chan int) {
	previous := time.Now()
	for {
		tick <- 1
		current := time.Now()
		elapsed := current.Sub(previous)
		previous = current

		log.Printf("step %f", elapsed)

		time.Sleep(250 * time.Millisecond)
	}
}

func updateCharge(vehicle *Vehicle) {
	charge := vehicle.C
	speed := vehicle.Speed

	vehicle.C = charge - (charge * 0.03 * speed)
}

func updateDirection(v *Vehicle, chargers []Charger) {

	charge := v.C
	if charge < 80 && charge > 10 {
		// compute distance to nearest charger
		// set direction
		for _, c := range chargers {
			dx := math.Pow((v.X - c.X), 2)
			dy := math.Pow((v.Y - c.Y), 2)
			dist := math.Sqrt(dx + dy)
			fmt.Printf("Dist to charger %s (%f,%f) from vehicle %s (%f,%f) is %f\n", c.Rego, c.X, c.Y, v.Rego, v.X, v.Y, dist)
		}

	} else if charge < 10 {
		fmt.Printf("Low energy!")
	}
}
func updateVehicle(track CircleTrack, vehicle *Vehicle) {

	speed := vehicle.Speed
	direction := vehicle.Direction
	start_point := vehicle.Point

	// object travellling at a rate of 1 unit on a curve radius 5 units
	// 1 unit/min, arc length
	// s  = theta r
	// 1  = theta (5)
	// 1/5 = theta (radians)

	angular_vel := speed / track.Radius
	// fmt.Printf("speed %f / radius %f is angular vel %f\n", speed, track.Radius, angular_vel)
	new_point := start_point + (angular_vel * direction)
	// fmt.Printf("start point is %f, angular_vel %f, direction %f = new_point %f\n", start_point, angular_vel, direction, new_point)
	if new_point > 2*math.Pi { // wrap around
		new_point = new_point - 2*math.Pi
	}
	if new_point < 0 { // wrap backaround
		new_point = new_point + 2*math.Pi
	}

	x, y := coord(track, new_point)
	vehicle.X = x
	vehicle.Y = y
	vehicle.Point = new_point

}

func updateCharger(track CircleTrack, charger *Charger) {
	start_point := charger.Point
	x, y := coord(track, start_point)
	charger.X = x
	charger.Y = y
}

func coord(track CircleTrack, rad float64) (float64, float64) {
	// parametric form
	// https://en.wikipedia.org/wiki/Circle#Equations
	x := track.Radius*math.Cos(rad) + track.j
	y := track.Radius*math.Sin(rad) + track.k
	return x, y
}

func main() {

	// simple signals
	done := make(chan int)
	tick := make(chan int)

	// game objects
	v_pipeline := make(chan *Vehicle)
	c_pipeline := make(chan *Charger)

	// debug to limit gameplay
	go limited(done, tick)
	// simple heartbeat
	go ticker(tick)

	// handle input
	go inputMapper(done)
	// perform state updates
	go updateState(done, tick, v_pipeline, c_pipeline)

	// render when data on pipeline
	go render(v_pipeline, c_pipeline)

	<-done
}
