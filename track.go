package main

import (
	"encoding/json"
	"fmt"
	"github.com/rooprob/chargesim/message"
	uuid "github.com/satori/go.uuid"
	"math"
	"math/rand"
	"sort"
)

type Track interface {
	Add(child Object)
	Childs() []Object
	Print(prefix string) string
	Tick()
	Render(render chan Object)
}

type StraightLineTrack struct {
	// track parameters to describe a circle
	Id     string
	Color  string
	Name   string
	Kind   int
	origin Points
	end    Points
	childs []Object
	points []Points
}

func NewStraightLineTrack(name string, origin Points, end Points) *StraightLineTrack {
	return &StraightLineTrack{
		Id:     uuid.Must(uuid.NewV4()).String(),
		Color:  generateColor(),
		Kind:   message.KindTrack,
		Name:   name,
		origin: origin,
		end:    end,
	}
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
	Id     string
	Color  string
	Name   string
	Kind   int
	origin Points
	radius float64
	childs []Object
	points []Points
	rads   []float64
	hints  []float64
}

func NewCircularTrack(name string, origin Points, radius float64) *CircularTrack {
	return &CircularTrack{
		Id:     uuid.Must(uuid.NewV4()).String(),
		Color:  generateColor(),
		Kind:   message.KindTrack,
		Name:   name,
		origin: origin,
		radius: radius,
	}
}

func (v *CircularTrack) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id     string  `json:"id"`
		Color  string  `json:"color"`
		Kind   int     `json:"kind"`
		Origin Points  `json:"origin"`
		Radius float64 `json:"radius"`
		Name   string  `json:"name"`
	}{
		Id:     v.Id,
		Color:  v.Color,
		Kind:   v.Kind,
		Origin: v.origin,
		Radius: v.radius,
		Name:   v.Name,
	})
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
	render <- self
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

func (self *CircularTrack) coords(rad float64) (float64, float64) {
	x := self.radius*math.Cos(rad) + self.origin.X
	y := self.radius*math.Sin(rad) + self.origin.Y
	return x, y
}

func (self *CircularTrack) Points() Points {
	return self.origin
}

func (self *CircularTrack) SetPoints(p Points) {
	self.origin = p
}
