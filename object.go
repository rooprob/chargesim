package main

import (
	"encoding/json"
	"fmt"
	"github.com/rooprob/chargesim/message"
	uuid "github.com/satori/go.uuid"
	"log"
	"math"
	"math/rand"
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

// Examples of objects are Vehicles, Chargers
type Object interface {
	Points() Points
	SetPoints(p Points)
	Print(prefix string) string
	Tick()
}

// A Vehicle
type Vehicle struct {
	Id                  string
	Kind                int
	Color               string
	Charge              float64
	Model, Name, Status string
	Velocity            float64
	points              Points
	hints               []*Hint
}

func NewVehicle(name, model, status string, charge float64) *Vehicle {
	return &Vehicle{
		Id:       uuid.Must(uuid.NewV4()).String(),
		Color:    generateColor(),
		Kind:     message.KindVehicle,
		Name:     name,
		Model:    model,
		Status:   status,
		Velocity: 1*rand.Float64() + 0.5, // very small (<.5) is treated as zero
		Charge:   charge,
	}
}

func (v Vehicle) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id       string  `json:"id"`
		Kind     int     `json:"kind"`
		Color    string  `json:"color"`
		Points   Points  `json:"points"`
		Charge   float64 `json:"charge"`
		Model    string  `json:"model"`
		Name     string  `json:"name"`
		Status   string  `json:"status"`
		Velocity float64 `json:"velocity"`
		Range    float64 `json:"range"`
		Hints    []*Hint `json:"hints"`
	}{
		Id:       v.Id,
		Kind:     v.Kind,
		Color:    v.Color,
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
	v.RouteToCharger() // may change state to Queued

	switch v.Status {
	case "drive":
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
	v.Charge = v.Charge + (100 * 0.01)
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

	if v.Status != "drive" {
		return
	}

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
	consumption = math.Abs(v.Velocity) / 5
	return (v.Charge / consumption)
}

func (v *Vehicle) Consume() {
	// XXX refactor with CalcRange()
	fmt.Printf("Charge is %.2f\n", v.Charge)
	v.Charge = v.Charge - (math.Abs(v.Velocity) / 5)
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
	Id                  string
	Kind                int
	Color               string
	points              Points
	Model, Name, Status string
	queue               []*Vehicle
}

func NewCharger(name, model, status string) *Charger {
	return &Charger{
		Id:     uuid.Must(uuid.NewV4()).String(),
		Color:  "#00ff00",
		Kind:   message.KindCharger,
		Name:   name,
		Model:  model,
		Status: status,
	}
}

func (c *Charger) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id          string `json:"id"`
		Kind        int    `json:"kind"`
		Color       string `json:"color"`
		Points      Points `json:"points"`
		Model       string `json:"model"`
		Name        string `json:"name"`
		Status      string `json:"status"`
		QueueLength int    `json:"queueLength"`
	}{
		Id:          c.Id,
		Kind:        c.Kind,
		Points:      c.Points(),
		Color:       c.Color,
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

func (c *Charger) Print(prefix string) string {
	j, err := json.Marshal(c)
	if err != nil {
		log.Printf("got error")
	}
	return fmt.Sprintf("%s <Charger: %s>\n", prefix, string(j))
}

func (c *Charger) String() string {
	return c.Print("/")
}
