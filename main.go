package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
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
	LinearVelocity     float64
	Direction          float64
}

type Charger struct {
	X, Y               float64
	Type, Rego, Status string
	Point              float64
	Queue              []*Vehicle
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
		{0, 0, 30.0, "tesla x", "AAA", "drive", 2 * math.Pi, 1, -1},
		//{0, 0, 90.0, "tesla x", "BBB", "drive", 2 * math.Pi, 0.75, 1},
	}
	var chargers = []Charger{
		{0, 0, "ch1", "A", "online", math.Pi / 3, []*Vehicle{}},
		{0, 0, "ch1", "B", "online", 3 * math.Pi / 2, []*Vehicle{}},
		{0, 0, "ch2", "C", "online", math.Pi, []*Vehicle{}},
	}

	for {
		// charger objects
		var charger *Charger
		for i := 0; i < len(chargers); i++ {
			// mutating record, so pass reference
			charger = &chargers[i]
			positionCharger(track, charger)
			// fmt.Printf("updated charger idx %d, rego: %s point %f, wiht x %f y %f\n", idx, r.Rego, r.Point, r.X, r.Y)
			c_pipeline <- charger
		}

		// vehicle objects
		var vehicle *Vehicle
		// for each item to compute the state for...
		for i := 0; i < len(vehicles); i++ {
			vehicle = &vehicles[i]

			switch vehicle.Status {
			case "drive":
				consumeCharge(vehicle)
				updateVehicleState(vehicle, chargers)
				positionVehicle(track, vehicle)
				fmt.Printf("updated vehicle rego: %s point %f, wiht x %f y %f\n", vehicle.Rego, vehicle.Point, vehicle.X, vehicle.Y)
				break
			case "parked":
				// do nothing
				break
			case "queued":
				processQueue(chargers)
				break
			case "charging":
				processQueue(chargers)
				break
			}
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

func consumeCharge(v *Vehicle) {
	charge := v.C

	v.C = charge - (100 * 0.01 * v.LinearVelocity)

	if v.C < 0.1 {
		v.LinearVelocity = 0
		v.Status = "parked"
	}
}

func reCharge(v *Vehicle) {
	charge := v.C

	v.Status = "charging"
	v.C = charge + (100 * 0.1)
}

func nearestCharger(v *Vehicle, chargers []Charger) (*Charger, float64) {

	var c *Charger
	var dist float64
	// compute distance to charger (linear distance)
	chargerPoints := make(map[float64]*Charger) // mapping charger rego to points (float64)
	for i := 0; i < len(chargers); i++ {
		c = &chargers[i]
		// find the shortest distance
		dx := math.Pow((v.X - c.X), 2)
		dy := math.Pow((v.Y - c.Y), 2)
		dist = math.Sqrt(dx + dy)
		chargerPoints[dist] = c
		fmt.Printf("Dist to charger %s (%.2f,%.2f) from vehicle %s (%.2f,%.2f) is %.2f units\n", c.Rego, c.X, c.Y, v.Rego, v.X, v.Y, dist)
	}
	// order
	var points []float64
	for point := range chargerPoints {
		points = append(points, point)
	}
	sort.Float64s(points)

	c = chargerPoints[points[0]]
	dist = points[0]
	fmt.Printf("Nearest charger to vehicle %s is charger %s %.2f units\n", v.Rego, c.Rego, dist)
	return c, dist
}

func processQueue(chargers []Charger) {
	var c *Charger
	var v *Vehicle
	for i := 0; i < len(chargers); i++ {
		c = &chargers[i]

		if len(c.Queue) > 0 {
			v = c.Queue[0]
			reCharge(v)
			if v.C >= 99.0 {
				fmt.Printf("Fully charged vehicle %s at charger %s %.2f\n", v.Rego, c.Rego)
				v.Status = "drive"
				_, c.Queue = c.Queue[0], c.Queue[1:]
			}
		}
	}
}

func updateChargeStation(c *Charger, v *Vehicle) {

	// check queue length
	// if len(c.Queue) == cap(c.Queue) {
	// error - queue full!
	//	return
	//}
	fmt.Printf("added %s to queue %s\n", v.Rego, c.Rego)
	c.Queue = append(c.Queue, v)
	v.Status = "queued"
}

func updateVehicleState(v *Vehicle, chargers []Charger) {

	var c *Charger
	var dist float64

	c, dist = nearestCharger(v, chargers)

	charge := v.C
	if charge < 5.0 {
		if dist < 1.0 {
			fmt.Printf("Arrived: vehicle %s at charger %s\n", v.Rego, c.Rego)
			updateChargeStation(c, v)
		}
	}
}

func positionVehicle(track CircleTrack, vehicle *Vehicle) {
	// Compute new x,y location for vehicle.

	// Based on speed and direction, given a starting point in rads.
	t := 1.0 // time tick
	v := vehicle.LinearVelocity
	direction := vehicle.Direction
	start_point := vehicle.Point

	// Angular Velocity Formulas
	// w = theta / t, where w = angular velocity, theta = position angle, and t = time.
	// w = s / rt, where s length of arc, r radius, t time.
	// w = v / t, where v is linear velocity, r radius
	w := v / track.Radius
	theta := w / t

	// Arc length forumula
	// theta = s / r, s length of subtended arc, r radius
	// s = theta / r
	s := theta / track.Radius

	// adding/subtracking radians from a start point.
	new_point := start_point + (w * direction)
	if new_point > 2*math.Pi { // wrap around
		new_point = new_point - 2*math.Pi
	}
	if new_point < 0 { // wrap backaround
		new_point = new_point + 2*math.Pi
	}

	x, y := coord(track, new_point)

	log.Printf("origin: (%.2f,%.2f) %.2f rads, w %.2f, s %.2f, v %.2f, theta %.2f  (%.2f,%.2f) %.2f rads\n", vehicle.X, vehicle.Y, vehicle.Point, w, s, v, theta, x, y, new_point)
	vehicle.X = x
	vehicle.Y = y
	vehicle.Point = new_point

}

func positionCharger(track CircleTrack, charger *Charger) {
	x, y := coord(track, charger.Point)
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
