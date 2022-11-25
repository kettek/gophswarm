package main

import (
	"bytes"
	"log"
	"math/rand"
	"os"
	"time"

	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	world *World
}

func (g *Game) Update() error {
	g.world.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.world.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 360
	//return 1280, 720
}

func (g *Game) Init(w, h int) {
	rand.Seed(time.Now().UnixMicro())
	// Load images
	b, err := os.ReadFile("res/gopher.png")
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	gopherImage = ebiten.NewImageFromImage(img)

	b, err = os.ReadFile("res/randy.png")
	if err != nil {
		panic(err)
	}
	img, _, err = image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	randyImage = ebiten.NewImageFromImage(img)

	b, err = os.ReadFile("res/ded.png")
	if err != nil {
		panic(err)
	}
	img, _, err = image.Decode(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	dedImage = ebiten.NewImageFromImage(img)

	b, err = os.ReadFile("res/pheromoneFader.kage")
	if err != nil {
		panic(err)
	}

	shader, err := ebiten.NewShader(b)
	if err != nil {
		panic(err)
	}
	pheromoneFadeShader = shader

	// Create world
	g.world = NewWorld(w, h)
	//g.world.SpawnColony(30+rand.Intn(w/2-60), 30+rand.Intn(h/2-60), 50)
	g.world.SpawnColony(w/2, h/2, 20)
	//g.world.SpawnFood(200, 200, 30)
	for i := 0; i < 100; i++ {
		g.world.SpawnFood(10+rand.Intn(w-20), 10+rand.Intn(h-20), float64(1+rand.Intn(20)))
	}
}

func main() {
	g := &Game{}

	g.Init(640, 360)

	ebiten.SetTPS(200)

	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("GophSwarm")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
