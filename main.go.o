package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	//"math/rand"
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

// limited controls the number of tickets for debugging and testing.
func limited(done chan int, tick chan int) {
	for i := 0; i < 360; i++ {
		<-tick
	}
	done <- 1
}

// render accepts new items to display (currently to screen) from a number of different channels
// items are pointers currently. Loop uses select to spin over pipes nonblocking.
// FIXME make polymorphic
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

			log.Printf("<Vehicle: %s> (Est. range %.2f units)", string(j), estimateVehicleRange(v))
		case c = <-c_pipeline:
			j, err := json.Marshal(c)
			if err != nil {
				log.Printf("got error")
			}

			log.Printf("<Charger: %s>", string(j))
		default:
			// prevent busy wait
			time.Sleep(250 * time.Millisecond)
		}
	}
}

// updateState state initialisation, state transition and update state
func updateState(done chan int, tick chan int, v_pipeline chan *Vehicle, c_pipeline chan *Charger) {

	// initialize objects
	track := CircleTrack{0.0, 0.0, 5.0}

	var vehicles = []Vehicle{
		{0, 0, 30.0, "tesla x", "AAA", "drive", 2 * math.Pi, 1, -1},          // 0deg
		{0, 0, 89.0, "tesla x", "BBB", "drive", math.Pi / 3, 0.75, 1},        // 60deg
		{0, 0, 79.0, "tesla s", "DDD", "drive", 5 / 6 * math.Pi, 0.75, 1},    // 150def
		{0, 0, 29.0, "tesla s", "EEE", "drive", 4 / 3 * math.Pi, 0.75, 1},    // 135deg
		{0, 0, 25.0, "leaf x", "XXX", "drive", math.Pi / 2, 0.65, 1},         // 90deg
		{0, 0, 33.0, "leaf y", "YYY", "drive", 2 / 3 * math.Pi / 2, 1.15, 1}, // 120deg
	}

	var chargers = []Charger{
		{0, 0, "ch1", "A", "online", math.Pi, []*Vehicle{}},         // 180deg
		{0, 0, "ch1", "B", "online", 3 / 2 * math.Pi, []*Vehicle{}}, // 270deg
		{0, 0, "ch2", "C", "online", 2 * math.Pi, []*Vehicle{}},     // 9deg
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
				// vehicle.LinearVelocity = 1.0 //  rand.Float64() + rand.Float64()
				consumeCharge(vehicle)
				routeToCharger(track, vehicle, chargers)
				positionVehicle(track, vehicle)
				// fmt.Printf("updated vehicle rego: %s point %f, wiht x %f y %f\n", vehicle.Rego, vehicle.Point, vehicle.X, vehicle.Y)
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
	// previous := time.Now()
	for {
		tick <- 1
		// current := time.Now()
		// elapsed := current.Sub(previous)
		// previous = current

		// log.Printf("step %f", elapsed)
		time.Sleep(250 * time.Millisecond)
	}
}

func estimateVehicleRange(v *Vehicle) float64 {

	var consumption float64
	// XXX refactor with consumeCharge
	// XXX make a exponential scale
	consumption = (100 * 0.01 * v.LinearVelocity / 1.9)

	return v.C / consumption
}

func consumeCharge(v *Vehicle) {
	charge := v.C

	v.C = charge - (100 * 0.01 * v.LinearVelocity / 1.9)

	if v.C < 0.1 {
		v.LinearVelocity = 0
		v.C = 0
		v.Status = "flat"
	}
}

func reCharge(v *Vehicle) {
	charge := v.C

	v.Status = "charging"
	v.C = charge + (100 * 0.1)
}

// Caculate the nearest charger (linear distance)
func nearestCharger(v *Vehicle, chargers []Charger) (*Charger, float64) {

	var c *Charger
	var linearDist float64
	// compute distance to charger (linear distance)
	chargerPoints := make(map[float64]*Charger) // mapping charger rego to points (float64)
	for i := 0; i < len(chargers); i++ {
		c = &chargers[i]
		// find the shortest distance
		dx := math.Pow((v.X - c.X), 2)
		dy := math.Pow((v.Y - c.Y), 2)
		linearDist = math.Sqrt(dx + dy)
		chargerPoints[linearDist] = c
		// fmt.Printf("Dist to charger %s (%.2f,%.2f) from vehicle %s (%.2f,%.2f) is %.2f units\n", c.Rego, c.X, c.Y, v.Rego, v.X, v.Y, dist)
	}
	// order
	var points []float64
	for point := range chargerPoints {
		points = append(points, point)
	}
	sort.Float64s(points)

	c = chargerPoints[points[0]]
	linearDist = points[0]
	log.Printf("Nearest: to vehicle %s is %s %.2f units\n", v.Rego, c.Rego, linearDist)
	return c, linearDist
}

// Calculate the next nearest by absolute radians
// Returns the two nearest
func nextNearestChargers(t CircleTrack, v *Vehicle, chargers []Charger) (*Charger, float64, *Charger, float64) {

	var c1, c2 *Charger
	var r1, r2 float64

	chargerRads := make(map[float64]*Charger) // mapping charger rego to points (float64)
	for i := 0; i < len(chargers); i++ {
		c1 = &chargers[i]

		// directional, with negative indicating a clockwise direction
		r := c1.Point - v.Point

		// correct for going beyond 180deg
		if r > math.Pi {
			r = (2*math.Pi - r) * -1
		}
		chargerRads[r] = c1

	}
	// order
	var keys []float64
	for r := range chargerRads {
		keys = append(keys, r)
	}

	// Sort smallest to largest
	sort.Slice(keys, func(i, j int) bool {
		return math.Abs(keys[i]) < math.Abs(keys[j])
	})

	if len(keys) == 1 {
		c1 = chargerRads[keys[0]]
		r1 = keys[0]

		log.Printf("Only Nearest: to vehicle %s is %s %.2f rads (%.2f units)\n", v.Rego, c1.Rego, r1, r1*t.Radius)
		return c1, r1, nil, 0
	}

	c1 = chargerRads[keys[0]]
	r1 = keys[0]
	c2 = chargerRads[keys[1]]
	r2 = keys[1]

	log.Printf("Nearest: to vehicle %s is %s %.2f rads (%.2f units), or %s %.2f rads (%.2f units)\n", v.Rego, c1.Rego, r1, r1*t.Radius, c2.Rego, r2, r2*t.Radius)
	return c1, r1, c2, r2

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
				log.Printf("Charged! vehicle %s at %s", v.Rego, c.Rego)
				v.Status = "drive"
				v.LinearVelocity = 1.0
				_, c.Queue = c.Queue[0], c.Queue[1:]
			}
		}
	}
}

// queue at this charger, taking slot to wait in line.
func queueAtCharger(c *Charger, v *Vehicle) {

	// check queue length
	// if len(c.Queue) == cap(c.Queue) {
	// error - queue full!
	//	return
	//}
	log.Printf("Queued! vehicle %s at %s", v.Rego, c.Rego)
	c.Queue = append(c.Queue, v)
	v.Status = "queued"
	v.LinearVelocity = 0.0
}

// compute whether to continue onto the next recharger, or if we wont make it,
// which direction to travel (-1 clockwise, +1 anticlockwise) in order to reach
// the nearest.
func routeToCharger(t CircleTrack, v *Vehicle, chargers []Charger) {

	var c1, c2 *Charger
	var r1, r2 float64
	var s1, s2 float64
	var ver float64

	if v.C > 50 {
		return
	}

	// at 50% charge, consider recharging
	c1, r1, c2, r2 = nextNearestChargers(t, v, chargers)

	ver = estimateVehicleRange(v)

	if len(chargers) > 1 {
		s2 = math.Abs(r2) * t.Radius
		if ver > s2 {
			// safe to continue to next charger
			log.Printf("Safe for vehicle %s to continue to %s (%.2f units, est. range %.2f)!", v.Rego, c2.Rego, s2, ver)
			return
		}
	} else {
		// all the way around
		s1 = math.Abs(r1)*t.Radius + (2 * math.Pi * t.Radius)
		if ver > s1 {
			// safe to continue to next charger
			log.Printf("Safe for vehicle %s to loop %.2f units back to %s (%.2f units, est. range %.2f)", v.Rego, s1, c1.Rego, s1, ver)
			return
		}

	}

	// wont make it to the next charger, so head to nearest
	if r1 > 0 {
		// positive difference, means the charger is further around on the positive direction (anticlockwise)
		if v.Direction > 0 {
			// keep going
			return
		} else {
			// we're heading away (clockwise)
			v.Direction = v.Direction * -1
		}
	} else {
		// negative difference means the chrager is further clockwiseit's further around anti-clockwise
		if v.Direction > 0 {
			v.Direction = v.Direction * -1
		} else {
			return
		}
	}
	// economy mode
	s1 = math.Abs(r1) * t.Radius
	if ver > s1 {
		// reduce speed
		log.Printf("ECO MODE! vehicle %s heading to %s", v.Rego, c1.Rego)
		v.LinearVelocity = v.LinearVelocity * .9
	}

	if s1 < 1.0 {
		log.Printf("Arrived! vehicle %s at %s", v.Rego, c1.Rego)
		v.LinearVelocity = 0.0
		queueAtCharger(c1, v)
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
	// Circumference of a circle
	// c = 2pi*r
	// theta = s / r, s length of subtended arc, r radius
	// s = theta * r
	s := theta * track.Radius

	// adding/subtracking radians from a start point.
	new_point := start_point + (w * direction)
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
	fmt.Printf("Vehicle %s position: (%.2f,%.2f) %.2f rads, w %.2f, s %.2f, v %.2f, theta %.2f\n", vehicle.Rego, vehicle.X, vehicle.Y, vehicle.Point, w, s, v, theta)

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
