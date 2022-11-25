package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var (
	gopherImage         *ebiten.Image
	dedImage            *ebiten.Image
	randyImage          *ebiten.Image
	pheromoneFadeShader *ebiten.Shader
)

type Gopher struct {
	id         int
	generation int
	x, y       float64
	kind       uint
	dead       bool
	direction  float64
	speed      float64
	desires    Desires
	lastAte    int
	targetMate int
}

type Desires map[int]float64

const (
	Hungy = iota
	Angy
	Horny
	Wander
)

func (d Desires) Most() (kind int, value float64) {
	for k, v := range d {
		if v > value {
			kind = k
			value = v
		}
	}
	return
}

const (
	Peckish = 0.4
	Hungry  = 0.8
	Stuffed = 0.1
)

const (
	HungyStep    = 0.0001
	HornyStep    = 0.002
	AngyStep     = 0.001
	DefaultSpeed = 0.1
	EatAmount    = 0.1
)

const (
	FoodPheromone = iota
	MatePheromone
	GopherPheromone
	HatePheromone
)

func (g *Gopher) Update(w *World) {
	if g.desires[Hungy] > 1 {
		g.dead = true
		return
	}

	g.lastAte++

	g.desires[Wander] += 0.00005

	// Regardless of all else, decrease angy by 0.001 per tick
	if g.desires[Angy] > 0 {
		g.desires[Angy] -= AngyStep
	}
	// Also reduce hungy and add same to horny.
	if g.desires[Hungy] > Hungry {
		if g.desires[Horny] < 1 {
			// I guess we convert food into getting randy.
			g.desires[Horny] += HornyStep
			g.desires[Hungy] += HungyStep
		}
	}

	if g.desires[Angy] > 0.5 {
	} else if g.desires[Hungy] > Hungry {
		// 1. Check if there is food within 8 radius.
		// 1.1 If so, eat it and place a "food" pheromone
		// 2. Check if there are pheromones for each 4x4 area around us. Go in the direction of the strongest one.
		// 3. Otherwise, wander randomly
		if v := w.FoodAt(int(g.x), int(g.y)); v > 0 {
			// Stop moving
			g.speed = 0
			// And eat!!!
			amount := EatAmount
			if v < EatAmount {
				amount = EatAmount - v
			}
			g.lastAte = -800
			w.EatAt(g.x, g.y, amount)
			//g.desires[Hungy] -= EatAmount
			g.desires[Hungy] = 0
			fmt.Println("eat", amount, g.desires[Hungy])
			w.PheromoneAt(FoodPheromone, g.x, g.y, 6, 0.001)
		} else if dir, none := w.ResourceDirectionNear(FoodPheromone, g.x, g.y, 32); !none {
			g.direction = dir + (-0.1 + rand.Float64()*0.2)
			g.speed = DefaultSpeed
		} else if dir, none := w.BestPheromoneDirectionNear(FoodPheromone, g.x, g.y, 4, 0.2); !none {
			g.direction = dir + (-0.1 + rand.Float64()*0.2)
			g.speed = DefaultSpeed
			//fmt.Println("hmm", dir)
		} else {
			g.speed = DefaultSpeed
			g.direction += -0.1 + rand.Float64()*0.2
		}
	} else if g.desires[Horny] > 0.8 {
		w.PheromoneAt(MatePheromone, g.x, g.y, 3, 0.02)
		//fmt.Println("horny")
		// 1. Check if there is another gopher within 8 radius.
		// 1.1 If so, mate with it and reduce our horny.
		// 2. Check if there are pheromones for each 4x4 area around us. Go in the direction of the strongest one.
		if g.targetMate != -1 {
			if g2 := w.Gopher(g.targetMate); g2 == nil {
				g.targetMate = -1
			} else {
				if g2.x >= g.x-2 && g2.x <= g.x+2 && g2.x >= g.y-2 && g2.y <= g.y+2 {
					// Let's do it to it!
					g.desires[Horny] = -1
					g2.desires[Horny] = -1
					w.BirthGopher(g, g2)
					g.targetMate = -1
					g2.targetMate = -1
					fmt.Println("BIRTH")
				} else {
					g.speed = DefaultSpeed * 3
					g.direction = math.Atan2(g2.y-g.y, g2.x-g.x)
				}
			}
		} else if id := w.NearestMateableGopherTo(g); id != -1 {
			g.targetMate = id
		} else if dir, none := w.BestPheromoneDirectionNear(MatePheromone, g.x, g.y, 8, 0.2); !none {
			g.direction = dir + (-0.1 + rand.Float64()*0.2)
			g.speed = DefaultSpeed * 2
		} else {
			g.speed = DefaultSpeed
			g.direction += -0.1 + rand.Float64()*0.2
		}
	} else {
		// wander...
		g.speed = DefaultSpeed
		g.direction += -0.1 + rand.Float64()*0.2
	}

	// Move towards our desired location and apply pheromones as necessary.
	if g.speed > 0 {
		x := math.Cos(g.direction) * g.speed
		y := math.Sin(g.direction) * g.speed

		if g.x+x < 0 || g.x+x >= float64(w.width) || g.y+y < 0 || g.y+y >= float64(w.height) {
			g.direction += math.Pi
			x = math.Cos(g.direction) * g.speed
			y = math.Sin(g.direction) * g.speed
		}

		g.x += x
		g.y += y

		// If we're satiated, mark our trail as such.
		/*if g.desires[Hungy] < Stuffed {
			w.PheromoneAt(FoodPheromone, g.x, g.y, 4, Hungry-g.desires[Hungy])
		}*/

		// Increase hunger if we move.
		g.desires[Hungy] += HungyStep
		g.desires[Wander] -= 0.00005
	} else {
		g.desires[Hungy] += HungyStep / 4
	}

	if g.lastAte < 0 {
		w.PheromoneAt(FoodPheromone, g.x, g.y, 3, (float64(g.lastAte)/-800)*0.005)
	}
	w.PheromoneAt(GopherPheromone, g.x, g.y, 2, 0.02)
}

type World struct {
	width, height  int
	pheromoneBytes []byte
	pheromoneMap   *ebiten.Image
	resourceMap    *ebiten.Image
	resourceBytes  []byte
	fadeCounter    int
	gophers        []Gopher
	deadGophers    []Gopher
	birthGophers   []Gopher
	gopherId       int
	topGeneration  int
	deaths         int
	births         int
}

func NewWorld(w, h int) *World {
	world := World{
		width:  w,
		height: h,
		pheromoneMap: ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		pheromoneBytes: make([]byte, w*h*4),
		resourceMap: ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{
			Unmanaged: true,
		}),
		resourceBytes: make([]byte, w*h*4),
	}

	return &world
}

func (w *World) Obliterate() {
	w.pheromoneMap.Dispose()
	w.resourceMap.Dispose()
	w.gophers = make([]Gopher, 0)
	w.deadGophers = make([]Gopher, 0)
}

// SpawnColony spawns a colony with a randomized spread at x, y
func (w *World) SpawnColony(x, y int, amount int) {
	for i := 0; i < amount; i++ {
		g := Gopher{
			id:        w.gopherId,
			x:         float64(x - 20 + rand.Intn(40)),
			y:         float64(y - 20 + rand.Intn(40)),
			desires:   make(Desires),
			speed:     0.1,
			direction: rand.Float64() * math.Pi * 2,
		}
		g.desires[Hungy] = Stuffed + rand.Float64()*0.1
		w.gophers = append(w.gophers, g)
		w.gopherId++
	}
}

func (w *World) BirthGopher(p1, p2 *Gopher) {
	g := Gopher{
		id:        w.gopherId,
		x:         (p1.x + p2.x) / 2,
		y:         (p1.y + p2.y) / 2,
		desires:   make(Desires),
		speed:     0.1,
		direction: rand.Float64() * math.Pi * 2,
	}
	if p1.generation > p2.generation {
		g.generation = p1.generation + 1
	} else {
		g.generation = p2.generation + 1
	}
	if w.topGeneration < g.generation {
		w.topGeneration = g.generation
	}
	g.desires[Hungy] = Stuffed + rand.Float64()*0.1
	w.birthGophers = append(w.birthGophers, g)
	w.gopherId++
	w.births++
}

func (w *World) SpawnFood(x, y int, radius float64) {
	w.placeCircle(w.resourceBytes, x, y, int(radius), color.RGBA{
		R: 0,
		G: 255,
		B: 0,
		A: 255,
	}, SetOp)
}

type Operation int

const (
	SetOp Operation = iota
	AddOp
	RemoveOp
)

func (w *World) placeCircle(b []byte, x0, y0 int, r int, c color.RGBA, op Operation) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y < r*r {
				w.setPixel(b, x0+x, y0+y, c, op)
			}
		}
	}
}

func (w *World) placeRing(b []byte, x0, y0, int, r int, c color.RGBA, op Operation) {
	x, y, dx, dy := r-1, 0, 1, 1

	err := dx - (r * 2)

	for x > y {
		w.setPixel(b, x0+x, y0+y, c, op)
		w.setPixel(b, x0+y, y0+x, c, op)
		w.setPixel(b, x0-y, y0+x, c, op)
		w.setPixel(b, x0-x, y0+y, c, op)
		w.setPixel(b, x0-x, y0-y, c, op)
		w.setPixel(b, x0-y, y0-x, c, op)
		w.setPixel(b, x0+y, y0-x, c, op)
		w.setPixel(b, x0+x, y0-y, c, op)
		if err <= 0 {
			y++
			err += dy
			dy += 2
		}
		if err > 0 {
			x--
			dx += 2
			err += dx - (r * 2)
		}
	}
}

func (w *World) getPixel(b []byte, x, y int) (c color.RGBA) {
	if x < 0 || x >= w.width || y < 0 || y >= w.height {
		return
	}
	c.R = b[(y*w.width+x)*4]
	c.G = b[(y*w.width+x)*4+1]
	c.B = b[(y*w.width+x)*4+2]
	c.A = b[(y*w.width+x)*4+3]
	return
}

func (w *World) setPixel(bs []byte, x, y int, c color.RGBA, op Operation) {
	if x < 0 || x >= w.width || y < 0 || y >= w.height {
		return
	}
	r := bs[(y*w.width+x)*4]
	g := bs[(y*w.width+x)*4+1]
	b := bs[(y*w.width+x)*4+2]
	a := bs[(y*w.width+x)*4+3]
	if op == AddOp {
		if int(r)+int(c.R) > 255 {
			r = 255
		} else {
			r = r + c.R
		}
		if int(g)+int(c.G) > 255 {
			g = 255
		} else {
			g = g + c.G
		}
		if int(b)+int(c.B) > 255 {
			b = 255
		} else {
			b = b + c.B
		}
		if int(a)+int(c.A) > 255 {
			a = 255
		} else {
			a = a + c.A
		}
	} else if op == RemoveOp {
		if int(r)-int(c.R) < 0 {
			r = 0
		} else {
			r = r - c.R
		}
		if int(g)-int(c.G) < 0 {
			g = 0
		} else {
			g = g - +c.G
		}
		if int(b)-int(c.B) < 0 {
			b = 0
		} else {
			b = b - c.B
		}
		if int(a)-int(c.A) < 0 {
			a = 0
		} else {
			a = a - c.A
		}
	} else {
		r = c.R
		g = c.G
		b = c.B
		a = c.A
	}
	bs[(y*w.width+x)*4] = r
	bs[(y*w.width+x)*4+1] = g
	bs[(y*w.width+x)*4+2] = b
	bs[(y*w.width+x)*4+3] = a
}

func (w *World) FoodAt(fx, fy int) float64 {
	for y := int(fy) - 4; y < int(fy)+4; y++ {
		for x := int(fx) - 4; x < int(fx)+4; x++ {
			if x < 0 || y < 0 || x >= w.width || y >= w.height {
				return 0
			}
			food := w.resourceBytes[(y*w.width+x)*4+1]
			if food > 10 {
				return float64(food) / 255
			}
		}
	}
	return 0
}

func (w *World) EatAt /*Joe's*/ (fx, fy float64, amount float64) {
	x := int(fx)
	y := int(fy)
	/*c := w.resourceMap.At(x, y)
	r, g, b, a := c.RGBA()
	g -= uint32(255 * amount)
	w.resourceMap.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})*/
	w.setPixel(w.resourceBytes, x, y, color.RGBA{0, 255, 0, 0}, RemoveOp)
}

func (w *World) ResourceDirectionNear(resource int, fx, fy float64, distance int) (dir float64, none bool) {
	for y := int(fy) - distance; y < int(fy)+distance; y++ {
		for x := int(fx) - distance; x < int(fx)+distance; x++ {
			if x < 0 || y < 0 || x >= w.width || y >= w.height {
				continue
			}
			g := w.pheromoneBytes[(y*w.width+x)*4+1]
			if g > 0 {
				return math.Atan2(float64(y)-fy, float64(x)-fx), false
			}
		}
	}
	return 0, true
}

func (w *World) PheromoneAt(pheromone int, fx, fy float64, radius float64, strength float64) {
	switch pheromone {
	case FoodPheromone:
		c := color.RGBA{0, uint8(strength * 255), 0, 0}
		w.placeCircle(w.pheromoneBytes, int(fx), int(fy), int(radius), c, AddOp)
	case MatePheromone:
		c := color.RGBA{uint8(strength * 255), 0, 0, 0}
		w.placeCircle(w.pheromoneBytes, int(fx), int(fy), int(radius), c, AddOp)
	case GopherPheromone:
		c := color.RGBA{0, 0, uint8(strength * 255), 0}
		w.placeCircle(w.pheromoneBytes, int(fx), int(fy), int(radius), c, AddOp)
	}
}

func (w *World) BestPheromoneDirectionNear(pheromone int, fx, fy float64, distance int, threshold float64) (dir float64, none bool) {
	bestX := 0.0
	bestY := 0.0
	bestValue := uint8(0)
	for y := int(fy) - distance; y < int(fy)+distance; y++ {
		for x := int(fx) - distance; x < int(fx)+distance; x++ {
			if x < 0 || y < 0 || x >= w.width || y >= w.height {
				continue
			}
			g := w.pheromoneBytes[(y*w.width+x)*4+1]
			//c := w.pheromoneMap.At(x, y)
			//_, g, _, _ := c.RGBA()
			if pheromone == FoodPheromone {
				if g > byte(threshold*255) && bestValue < uint8(g) {
					bestX = float64(x)
					bestY = float64(y)
					bestValue = uint8(g)
				}
			}
		}
	}
	if bestValue == 0 {
		return 0, true
	}
	return math.Atan2(bestY-fy, bestX-fx), false
}

func (w *World) NearestMateableGopherTo(g *Gopher) int {
	for _, g2 := range w.gophers {
		if g2.id == g.id {
			continue
		}
		if kind, _ := g2.desires.Most(); kind != Horny {
			continue
		}
		if g2.x >= g.x-60 && g2.x <= g.x+60 && g2.y >= g.y-60 && g2.y <= g.y+60 {
			return g2.id
		}
	}
	return -1
}

func (w *World) Gopher(id int) *Gopher {
	for i, g := range w.gophers {
		if g.id == id {
			return &w.gophers[i]
		}
	}
	return nil
}

// Update it, son.
func (w *World) Update() {
	// Fade pheromones
	w.fadeCounter++
	if w.fadeCounter > 10 {
		for i := 0; i < len(w.pheromoneBytes); i += 4 {
			if w.pheromoneBytes[i] > 0 {
				w.pheromoneBytes[i]--
			}
			if w.pheromoneBytes[i+1] > 0 {
				w.pheromoneBytes[i+1]--
			}
			if w.pheromoneBytes[i+2] > 0 {
				w.pheromoneBytes[i+2]--
			}
		}
		w.fadeCounter = 0
	}

	t := w.gophers[:0]
	for _, g := range w.gophers {
		g.Update(w)
		if !g.dead {
			t = append(t, g)
		} else {
			g.speed = 1.0
			w.deadGophers = append(w.deadGophers, g)
			w.deaths++
		}
	}
	w.gophers = t

	for _, g := range w.birthGophers {
		w.gophers = append(w.gophers, g)
	}
	w.birthGophers = make([]Gopher, 0)

	t = w.deadGophers[:0]
	for _, g := range w.deadGophers {
		g.speed -= 0.001
		if g.speed > 0 {
			t = append(t, g)
		}
	}
	w.deadGophers = t

	w.pheromoneMap.WritePixels(w.pheromoneBytes)
	w.resourceMap.WritePixels(w.resourceBytes)
}

func (w *World) Draw(screen *ebiten.Image) {
	//screen.Fill(color.RGBA{124, 94, 66, 255})
	// Draw pheromone map
	screen.DrawImage(w.pheromoneMap, nil)
	// Draw food map
	screen.DrawImage(w.resourceMap, nil)
	for _, g := range w.deadGophers {
		opts := ebiten.DrawImageOptions{}
		opts.ColorM.Scale(1.0, 1.0, 1.0, g.speed)
		opts.GeoM.Translate(g.x-float64(dedImage.Bounds().Dx()/2), g.y-float64(dedImage.Bounds().Dy()/2))
		screen.DrawImage(dedImage, &opts)
	}
	for _, g := range w.gophers {
		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Translate(g.x-float64(gopherImage.Bounds().Dx()/2), g.y-float64(gopherImage.Bounds().Dy()/2))

		if g.desires[Horny] > 0.8 {
			screen.DrawImage(randyImage, &opts)
		} else {
			screen.DrawImage(gopherImage, &opts)
		}
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nDeaths: %d\nBirths: %d\nGenerations: %d", ebiten.ActualTPS(), ebiten.ActualFPS(), w.deaths, w.births, w.topGeneration))
}
