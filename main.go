package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

type Points struct {
	X, Y float64
}

type Hint struct {
	TrackLength float64
	Dist        float64
	Vector      float64
	Charger     *Charger
	Range       float64
	InRange     bool
	NextRange   bool
}

type Track interface {
	Add(child Object)
	Childs() []Object
	Print(prefix string) string
	Tick()
	Render(render chan Object)
}

type StraightLineTrack struct {
	// track parameters to describe a circle
	origin Points
	end    Points

	Name   string
	childs []Object
	points []Points
}

// Adds an element to the tree branch
func (self *StraightLineTrack) Add(child Object) {
	self.childs = append(self.childs, child)
}

// Returns the child elements
func (self *StraightLineTrack) Childs() []Object {
	return self.childs
}

// Returns the child elements to render
func (self *StraightLineTrack) Render(render chan Object) {
	for _, val := range self.Childs() {
		render <- val
	}
}

// Returns a listing of the tree
func (self *StraightLineTrack) Print(prefix string) string {
	result := fmt.Sprintf("%s/%s\n", prefix, self.Name)
	for _, val := range self.Childs() {
		result += val.Print(fmt.Sprintf("%s/%s", prefix, self.Name))
	}
	return result
}

func (self *StraightLineTrack) String() string {
	return self.Print("/")
}

func (self *StraightLineTrack) Tick() {
	for i := 0; i < len(self.childs); i++ {
		self.childs[i].Tick()
	}
}

type CircularTrack struct {
	// track parameters to describe a circle
	origin Points
	radius float64

	Name   string
	childs []Object
	points []Points
	rads   []float64
	hints  []float64
}

// Adds an element to the tree branch
func (self *CircularTrack) Add(child Object) {
	self.childs = append(self.childs, child)
}

// Returns the child elements
func (self *CircularTrack) Childs() []Object {
	return self.childs
}

// Returns the child elements to render
func (self *CircularTrack) Render(render chan Object) {
	for _, val := range self.Childs() {
		render <- val
	}
}

// Returns a listing of the tree
func (self *CircularTrack) Print(prefix string) string {
	result := fmt.Sprintf("%s/%s\n", prefix, self.Name)
	for _, val := range self.Childs() {
		result += val.Print(fmt.Sprintf("%s/%s", prefix, self.Name))
	}
	return result
}

func (self *CircularTrack) String() string {
	return self.Print("/")
}

func (self *CircularTrack) RandomizeObjects() {
	rand.Seed(42)

	// create a new slice for child radians
	rads := make([]float64, len(self.childs))
	for idx, _ := range self.childs {
		theta := rand.Float64() * 2 * math.Pi
		rads[idx] = theta
	}
	self.rads = rads
	self.ComputeNewCoords()
}

func (self *CircularTrack) Tick() {
	fmt.Println("Track Tick")
	for i := 0; i < len(self.childs); i++ {
		self.childs[i].Tick()
	}
	self.ComputeNewPositions()
	self.ComputeNewCoords()
	self.ComputeHints()
}

func (self *CircularTrack) ComputeNewPositions() {
	// only the Vehicles
	vi := make(map[int]*Vehicle, len(self.childs))
	for i := 0; i < len(self.childs); i++ {
		switch v := self.childs[i].(type) {
		case *Vehicle:
			vi[i] = v
			break
		}
	}
	for i := range vi {
		t := 1.0 // time tick
		v := vi[i].Velocity
		p := self.rads[i]

		w := v / self.radius
		theta := w / t
		// s := theta * self.radius

		self.rads[i] = p + theta
		if self.rads[i] > 2*math.Pi {
			self.rads[i] = self.rads[i] - 2*math.Pi
		} else if self.rads[i] < 0 {
			self.rads[i] = self.rads[i] + 2*math.Pi
		}
	}
}

func (self *CircularTrack) ComputeNewCoords() {
	points := make([]Points, len(self.childs))
	for idx, _ := range self.childs {
		x, y := self.coords(self.rads[idx])
		p := Points{
			X: x,
			Y: y,
		}
		points[idx] = p
		self.childs[idx].SetPoints(p)
	}
	self.points = points
}

func (self *CircularTrack) Direction(theta, f float64) float64 {

	// Anticlockwise is a positive increment in radians
	// from 0 at 90degress, upto 2*pi back at the start.
	// Vehicles transiting anti-clockwise are +ve,
	// clockwise -ve
	if math.Signbit(theta) {
		// car is at 10o'clock, charger at 1o'clock
		if math.Signbit(f) {
			// car is going clockwise
			// keep going
			fmt.Println("Keep going...")
			return -1
		} else {
			// car is going anticlockwise
			// gone too far, turn back
			fmt.Println("Turning back...")
			return -1
		}
	} else {
		// charger is at 10o'clock, vehicle at 1o'clock
		if math.Signbit(f) {
			// gone too far,
			// turn back
			fmt.Println("Turning back...")
			return 1
		} else {
			// car is going anticlockwise,
			// continue
			fmt.Println("Keep going...")
			return 1
		}
	}
}

func (self *CircularTrack) Length(theta float64) float64 {
	return math.Abs(theta) * self.radius
}

// InRange this transit pass and next transit pass
func (self *CircularTrack) InRange(l, r float64) bool {
	return l < r
}

// Find the nearest Chargers
// Uses data from Track size/shape, list of Chargers and combines with a Vehicle state.
// Provides a Hints struture indicating next charging stop and range.
// XXX move from Track type
func (self *CircularTrack) ComputeHints() {
	// TODO look into composition instead of this.

	fmt.Println("ComputeHints")
	// map of supported types
	vi := make(map[int]*Vehicle, len(self.childs))
	ci := make(map[int]*Charger, len(self.childs))
	for i := 0; i < len(self.childs); i++ {
		switch v := self.childs[i].(type) {
		case *Vehicle:
			vi[i] = v
			break
		case *Charger:
			ci[i] = v
			break
		}
	}

	// for all Vehicles, provide a Hints structure
	// NOTE we can modify map inplace and effect self.childs by reference.
	var vr, cr, theta float64
	var v *Vehicle
	for vdx := range vi {
		v = vi[vdx]

		// Find nearest Chargers
		thetas := make(map[float64]*Charger, len(ci))
		// prepare sorted list of theta
		vr = self.rads[vdx]
		for cdx := range ci {
			cr = self.rads[cdx]

			// directional, -ve indicating clockwise
			theta = cr - vr

			// correct for going beyond pi (180deg)
			if theta > math.Pi {
				theta = (2*math.Pi - theta) * -1
			}
			// track map of radians and Charger
			thetas[theta] = ci[cdx]
		}
		// sort smallest to largest by Abs value
		thetaKeys := make([]float64, 0, len(ci))
		for kt := range thetas {
			thetaKeys = append(thetaKeys, kt)
		}
		sort.Slice(thetaKeys, func(i, j int) bool {
			return math.Abs(thetaKeys[i]) < math.Abs(thetaKeys[j])
		})

		// Ordered list of hints, per vehicle, sized by chargers
		hints := make([]*Hint, 0, len(ci))
		// prepare a list of structures
		for kt := range thetaKeys {
			hints = append(hints, &Hint{
				TrackLength: self.Length(2 * math.Pi),
				Dist:        self.Length(thetaKeys[kt]),
				Vector:      self.Direction(thetaKeys[kt], v.Velocity),
				Range:       v.CalcRange(),
				InRange:     self.InRange(self.Length(math.Abs(thetaKeys[kt])), v.CalcRange()),
				NextRange:   self.InRange(self.Length(2*math.Pi+math.Abs(thetaKeys[kt])), v.CalcRange()),
				Charger:     thetas[thetaKeys[kt]],
			})
		}
		v.SetHints(hints)
	}
}

func (self CircularTrack) coords(rad float64) (float64, float64) {
	x := self.radius*math.Cos(rad) + self.origin.X
	y := self.radius*math.Sin(rad) + self.origin.Y
	return x, y
}

// Examples of objects are Vehicles, Chargers
type Object interface {
	Points() Points
	SetPoints(p Points)
	Print(prefix string) string
	Tick()
}

// A Vehicle
type Vehicle struct {
	points              Points
	Charge              float64
	Model, Name, Status string
	Velocity            float64
	hints               []*Hint
}

func (v Vehicle) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Points   Points  `json:"Points"`
		Charge   float64 `json:"Charge"`
		Model    string  `json:"Model"`
		Name     string  `json:"Name"`
		Status   string  `json:"Status"`
		Velocity float64 `json:"Velocty"`
		Range    float64 `json:"Range"`
		Hints    []*Hint `json:"Hints"`
	}{
		Points:   v.Points(),
		Charge:   v.Charge,
		Model:    v.Model,
		Name:     v.Name,
		Status:   v.Status,
		Velocity: v.Velocity,
		Range:    v.CalcRange(),
		Hints:    v.Hints(),
	})
}

func (v *Vehicle) SetPoints(p Points) {
	v.points = p
}

func (v *Vehicle) Points() Points {
	return v.points
}

func (v *Vehicle) SetHints(p []*Hint) {
	v.hints = p
}

func (v *Vehicle) Hints() []*Hint {
	return v.hints
}

func (v *Vehicle) Tick() {
	// tick
	switch v.Status {
	case "drive":
		v.RouteToCharger()
		v.Drive()
		break
	case "parked":
		// do nothing
		break
	case "queued":
		// do nothing
		break
	case "charging":
		// do nothing
		break
	case "flat":
		// distress call!
		break
	}
}

// State setting
func (v *Vehicle) Flat() {
	v.Status = "flat"
	v.Velocity = 0.0
	v.Charge = 0.0
}

func (v *Vehicle) Drive() {
	v.Status = "drive"
	if len(v.hints) > 0 && v.hints[0].InRange == false {
		v.EcoMode()
	} else if len(v.hints) > 0 && v.hints[0].InRange == true {
		v.Velocity = v.Velocity * 1.01
	}
	v.Consume()
}

func (v *Vehicle) Charging() {
	v.Status = "charging"
	v.Charge = v.Charge + (100 * 0.02)
	if v.Charge > 100.0 {
		v.Drive()
	}
}

func (v *Vehicle) Queued() {
	v.Status = "queued"
	v.Velocity = 0.0
}

func (v *Vehicle) Parked() {
	v.Status = "parked"
	v.Velocity = 0.0
}

func (v *Vehicle) EcoMode() {
	if v.Velocity < 0.4 {
		v.Velocity = 0.4
	} else if v.Velocity > 1.0 {
		v.Velocity = v.Velocity * 0.9
	}
	fmt.Println("ECO mode")
}

// Process Hints data to determine whether the stop and recharge, or go on.
func (v *Vehicle) RouteToCharger() {

	if len(v.hints) > 1 {
		if v.hints[1].InRange {
			return
		}
	} else if len(v.hints) == 1 {
		if v.hints[0].NextRange {
			return
		}
	} else if len(v.hints) == 0 {
		return
	}

	// Take the Hint
	// We wont make it to the next charger, so head to nearest
	if math.Signbit(v.hints[0].Vector) != math.Signbit(v.Velocity) {
		v.Velocity = v.Velocity * -1
	}
	if v.hints[0].Dist < 1.0 {
		v.hints[0].Charger.Add(v)
	}
}

func (v *Vehicle) CalcRange() float64 {
	if v.Charge < 0.1 {
		return 0.0
	}
	if math.Abs(v.Velocity) < 0.1 {
		return 0.0
	}
	var consumption float64
	// XXX refactor with Consume()
	consumption = (100 * 0.01 * math.Abs(v.Velocity) / 1.9)
	return (v.Charge / consumption)
}

func (v *Vehicle) Consume() {
	// XXX refactor with CalcRange()
	fmt.Printf("Charge is %.2f\n", v.Charge)
	v.Charge = v.Charge - (100 * 0.01 * math.Abs(v.Velocity) / 1.9)
	if v.Charge < 0.1 {
		v.Flat()
	}
}

func (v *Vehicle) Print(prefix string) string {
	j, err := json.Marshal(v)
	if err != nil {
		log.Printf("got error")
	}
	return fmt.Sprintf("%s <Vehicle: %s>\n", prefix, string(j))
}

func (v *Vehicle) String() string {
	return v.Print("/")
}

// A Charger
type Charger struct {
	points              Points
	Model, Name, Status string
	queue               []*Vehicle
}

func (c *Charger) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Points      Points `json:"Points"`
		Model       string `json:"Model"`
		Name        string `json:"Name"`
		Status      string `json:"Status"`
		QueueLength int    `json:"QueueLength"`
	}{
		Points:      c.Points(),
		Model:       c.Model,
		Name:        c.Name,
		Status:      c.Status,
		QueueLength: len(c.Queue()),
	})
}

// Adds an element to the tree branch
func (c *Charger) Add(child *Vehicle) {

	if len(c.queue) >= 3 {
		return
	}
	fmt.Println("adding to Queue")
	c.queue = append(c.queue, child)
	child.Queued()
}

// Returns the child elements
func (c *Charger) Queue() []*Vehicle {
	return c.queue
}

func (c *Charger) SetPoints(p Points) {
	c.points = p
}

func (c *Charger) Points() Points {
	return c.points
}

func (c *Charger) ProcessQueue() {
	if len(c.queue) > 0 {
		if c.queue[0].Charge < 100 {
			c.queue[0].Charging()
		} else {
			c.queue[0].Drive()
			_, c.queue = c.queue[0], c.queue[1:]
		}
	} else {
		fmt.Println("Nothing queued")
	}
}

func (c *Charger) Tick() {
	// lifecycle event
	// process queue
	c.ProcessQueue()
	// increase/decrease random amount
}

func (c Charger) Print(prefix string) string {
	j, err := json.Marshal(c)
	if err != nil {
		log.Printf("got error")
	}
	return fmt.Sprintf("%s <Charger: %s>\n", prefix, string(j))
}

func (c Charger) String() string {
	return c.Print("/")
}

func NewVehicle(name, model, status string, charge float64) *Vehicle {
	return &Vehicle{
		Name:     name,
		Model:    model,
		Status:   status,
		Velocity: 1*rand.Float64() + 0.5, // very small (<.5) is treated as zero
		Charge:   charge,
	}
}

func NewCharger(name, model, status string) *Charger {
	return &Charger{
		Name:   name,
		Model:  model,
		Status: status,
	}
}

func NewCircularTrack(name string, origin Points, radius float64) *CircularTrack {
	return &CircularTrack{
		Name:   name,
		origin: origin,
		radius: radius,
	}
}

func NewStraightLineTrack(name string, origin Points, end Points) *StraightLineTrack {
	return &StraightLineTrack{
		Name:   name,
		origin: origin,
		end:    end,
	}
}

// limited controls the number of tickets for debugging and testing.
func limited(done chan int, tick chan int) {
	for i := 0; i < 360; i++ {
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
		time.Sleep(250 * time.Millisecond)
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

func handleRender(tick chan int, render chan Object) {
	// previous := time.Now()
	var v Object
	for {
		v = <-render
		fmt.Println(v)
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

	t1 := NewCircularTrack("T", Points{0.0, 0.0}, 20.0)
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
	go limited(done, tick)
	go handleInput(done)
	go handleRuntime(t1, tick, render)
	go handleRender(tick, render)
	<-done
}
