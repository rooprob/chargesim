package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
)

type Points struct {
	X, Y float64
}

type Track interface {
	Add(child Object)
	Name() string
	Childs() []Object
	Print(prefix string) string
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
func (self StraightLineTrack) Childs() []Object {
	return self.childs
}

// Returns a listing of the tree
func (self StraightLineTrack) Print(prefix string) string {
	result := fmt.Sprintf("%s/%s\n", prefix, self.Name)
	for _, val := range self.Childs() {
		result += val.Print(fmt.Sprintf("%s/%s", prefix, self.Name))
	}
	return result
}

func (self StraightLineTrack) String() string {
	return self.Print("/")
}

type CircularTrack struct {
	// track parameters to describe a circle
	origin Points
	radius float64

	Name   string
	childs []Object
	points []Points
	rads   []float64
}

// Adds an element to the tree branch
func (self *CircularTrack) Add(child Object) {
	self.childs = append(self.childs, child)
}

// Returns the child elements
func (self CircularTrack) Childs() []Object {
	return self.childs
}

// Returns a listing of the tree
func (self CircularTrack) Print(prefix string) string {
	result := fmt.Sprintf("%s/%s\n", prefix, self.Name)
	for _, val := range self.Childs() {
		result += val.Print(fmt.Sprintf("%s/%s", prefix, self.Name))
	}
	return result
}

func (self CircularTrack) String() string {
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
	self.ComputeAllFromRads()
}

func (self *CircularTrack) ComputeAllFromRads() {
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
}

// A Vehicle
type Vehicle struct {
	points              Points
	Capacity            float64
	Model, Name, Status string
	Velocity            float64
}

func (v Vehicle) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Points   Points  `json:"Points"`
		Capacity float64 `json:"Capacity"`
		Model    string  `json:"Model"`
		Name     string  `json:"Name"`
		Status   string  `json:"Status"`
		Velocity float64 `json:"Velocty"`
		Range    float64 `json:"Range"`
	}{
		Points:   v.Points(),
		Capacity: v.Capacity,
		Model:    v.Model,
		Name:     v.Name,
		Status:   v.Status,
		Velocity: v.Velocity,
		Range:    v.CalcRange(),
	})
}

func (v *Vehicle) SetPoints(p Points) {
	v.points = p
}

func (v Vehicle) Points() Points {
	return v.points
}

func (v Vehicle) CalcRange() float64 {
	if v.Capacity < 0.1 {
		return 0.0
	}
	if math.Abs(v.Velocity) < 0.1 {
		return 0.0
	}
	var consumption float64
	// XXX refactor with consumeCharge
	// XXX make a exponential scale
	consumption = (100 * 0.01 * math.Abs(v.Velocity) / 1.9)
	return (v.Capacity / consumption)
}

func (v Vehicle) Print(prefix string) string {
	j, err := json.Marshal(v)
	if err != nil {
		log.Printf("got error")
	}
	return fmt.Sprintf("%s <Vehicle: %s>\n", prefix, string(j))
}

func (v Vehicle) String() string {
	return v.Print("/")
}

// A Charger
type Charger struct {
	points              Points `json:"Points"`
	Model, Name, Status string
	queue               []Vehicle `json:"Queue"`
}

func (c Charger) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Points Points    `json:"Points"`
		Model  string    `json:"Model"`
		Name   string    `json:"Name"`
		Status string    `json:"Status"`
		Queue  []Vehicle `json:"Queue"`
	}{
		Points: c.Points(),
		Model:  c.Model,
		Name:   c.Name,
		Status: c.Status,
		Queue:  c.Queue(),
	})
}

// Adds an element to the tree branch
func (c *Charger) Add(child Vehicle) {
	c.queue = append(c.queue, child)
}

// Returns the child elements
func (c Charger) Queue() []Vehicle {
	return c.queue
}
func (c Charger) SetPoints(p Points) {
	c.points = p
}

func (c Charger) Points() Points {
	return c.points
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

func NewVehicle(name, model, status string) *Vehicle {
	return &Vehicle{
		Name:     name,
		Model:    model,
		Status:   status,
		Velocity: 1*rand.Float64() + 0.5,   // very small (<.5) is treated as zero
		Capacity: 100*rand.Float64() + 0.5, // very small (<.5) is treated as zero
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

func main() {

	v1 := NewVehicle("AAA", "Model X", "drive")
	v2 := NewVehicle("BBB", "Model X", "drive")
	v3 := NewVehicle("CCC", "Model S", "drive")
	v4 := NewVehicle("ZZZ", "Leaf", "drive")

	c1 := NewCharger("A", "t1", "online")
	c2 := NewCharger("B", "t1", "online")
	c3 := NewCharger("C", "t2", "online")

	t1 := NewCircularTrack("T", Points{0.0, 0.0}, 5.0)
	t1.Add(v1)
	t1.Add(v2)
	t1.Add(v3)
	t1.Add(v4)
	t1.Add(c1)
	t1.Add(c2)
	t1.Add(c3)

	t1.RandomizeObjects()

	fmt.Printf("%s\n", t1.Print(""))

	t2 := NewStraightLineTrack("T", Points{2, 2}, Points{4, 4})
	t2.Add(v1)
	t2.Add(c1)
	fmt.Printf("%s\n", t2.Print(""))

}
