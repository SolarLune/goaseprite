# goaseprite

[pkg.go.dev link](https://pkg.go.dev/github.com/solarlune/goaseprite)

Goaseprite is a JSON loader for Aseprite files for Golang.

## How To Use

Usage is pretty straightforward. You export a sprite sheet and its corresponding JSON data file from Aseprite (Ctrl+E). The values should be set to Hash with Frame Tags and Slices (optionally) on.

Then you'll want to load the Aseprite data. To do this, you'll call `goaseprite.Open()` with a string argument of where to find the Aseprite JSON data file, or manually pass the bytes to `goaseprite.Read()`. From this, you'll get a `*goaseprite.File`, which represents an Aseprite file. It's from here that you control your animation. You can just call `File.Play()` to play a tag / animation, and use the `File.Update()` function with an argument of delta time (the time between the previous frame and the current one) to update the animation. Call `File.CurrentFrame()` to get the current frame, which gives you the X and Y position of the current frame on the sprite sheet. Assuming a tag with a blank name ("") doesn't exist in your Aseprite file, `goaseprite` will create a default animation with that name, allowing you to easily play all of the frames in sequence.

Here'a quick example, using [ebiten](https://ebiten.org/) for rendering:

```go
package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/solarlune/goaseprite"

	_ "image/png"
)

type Game struct {
	Sprite *goaseprite.File
	Img    *ebiten.Image
}

func NewGame() *Game {

	game := &Game{
		Sprite: goaseprite.Open("16x16Deliveryman.json"),
	}

	img, _, err := ebitenutil.NewImageFromFile(game.Sprite.ImagePath)
	if err != nil {
		panic(err)
	}

    game.Sprite.Play("walk")

	game.Img = img

	return game

}

func (game *Game) Update() error {

	game.Sprite.Update(float32(1.0 / 60.0))

	return nil
}

func (game *Game) Draw(screen *ebiten.Image) {

	opts := &ebiten.DrawImageOptions{}

	srcX := 0
	srcY := 0

    // Could be nil if you don't immediately start playing an animation above
	if game.Sprite.CurrentFrame() != nil {
		srcX = game.Sprite.CurrentFrame().X
		srcY = game.Sprite.CurrentFrame().Y
	}

	sub := game.Img.SubImage(image.Rect(srcX, srcY, srcX+16, srcY+16))

	screen.DrawImage(sub.(*ebiten.Image), opts)

}

func (game *Game) Layout(w, h int) (int, int) { return 320, 180 }

func main() {

	ebiten.RunGame(NewGame())

}


```

## Additional Notes

As for dependencies, GoAseprite makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 
