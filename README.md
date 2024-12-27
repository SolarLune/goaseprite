# goaseprite

[pkg.go.dev link](https://pkg.go.dev/github.com/solarlune/goaseprite)

Goaseprite is a JSON loader for Aseprite files for Golang.

## How To Use

Usage is pretty straightforward. You export a sprite sheet and its corresponding JSON data file from Aseprite (Ctrl+E). The values should be set to Hash with Frame Tags and Slices (optionally) on.

Then you'll want to load the Aseprite data. To do this, you'll call `goaseprite.Open()` with a string argument of where to find the Aseprite JSON data file, or manually pass the bytes to `goaseprite.Read()`. From this, you'll get a `*goaseprite.File`, which represents an Aseprite file. It's from here that you control your animation.

You can call `File.Play()` to play a tag / animation, and use the `File.Update()` function with an argument of delta time (the time between the previous frame and the current one) to update the animation. Call `File.CurrentFrame()` to get the current frame, which gives you the X and Y position of the current frame on the sprite sheet. Assuming a tag with a blank name ("") doesn't exist in your Aseprite file, `goaseprite` will create a default animation with that name, allowing you to easily play all of the frames in sequence.

Here'a quick example, using [ebiten](https://ebiten.org/) for rendering:

```go
package main

import (
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/solarlune/goaseprite"

	_ "image/png"
)

type Game struct {
	Sprite    *goaseprite.File
	AsePlayer *goaseprite.Player
	Img       *ebiten.Image
}

func NewGame() *Game {

	sprite, err := goaseprite.Open("16x16Deliveryman.json", os.DirFS("."))
	if err != nil {
		panic(err)
	}
	game := &Game{
		Sprite: sprite,
	}

	game.AsePlayer = game.Sprite.CreatePlayer()

	// There are four callback functions that you can use to watch for changes to the internal state of the *File.

	// OnLoop is called when the animation is finished and a loop is completed; for ping-pong, it happens on a full revolution (after going forwards and then backwards).
	// game.Sprite.OnLoop = func() { fmt.Println("loop") }

	// OnFrameChange is called when the sprite's frame changes.
	// game.Sprite.OnFrameChange = func() { fmt.Println("frame change") }

	// OnTagEnter is called when the File enters a new Tag (i.e. if you play an animation of a sword being slashed, you can make this callback watch for a tag that indicates when a corresponding sound should play).
	// game.Sprite.OnTagEnter = func(tag *goaseprite.Tag) { fmt.Println("entered: ", tag.Name) }

	// OnTagExit is called when the File leaves the current Tag.
	// game.Sprite.OnTagExit = func(tag *goaseprite.Tag) { fmt.Println("exited: ", tag.Name) }

	img, _, err := ebitenutil.NewImageFromFile(game.Sprite.ImagePath)
	if err != nil {
		panic(err)
	}

	// game.Sprite.PlaySpeed = 2

	game.Img = img

	ebiten.SetWindowTitle("goaseprite example")
	ebiten.SetWindowResizable(true)

	game.AsePlayer.Play("idle")

	return game

}

func (game *Game) Update() error {

	if ebiten.IsKeyPressed(ebiten.Key1) {
		game.AsePlayer.Play("idle")
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		game.AsePlayer.Play("walk")
	} else if ebiten.IsKeyPressed(ebiten.Key3) {
		game.AsePlayer.Play("") // Calling Play() with a blank string will play the full animation (similar to playing an animation in Aseprite without any tags selected).
	}

	game.AsePlayer.Update(float32(1.0 / 60.0))

	return nil
}

func (game *Game) Draw(screen *ebiten.Image) {

	opts := &ebiten.DrawImageOptions{}

	sub := game.Img.SubImage(image.Rect(game.AsePlayer.CurrentFrameCoords()))

	screen.DrawImage(sub.(*ebiten.Image), opts)

}

func (game *Game) Layout(w, h int) (int, int) { return 320, 180 }

func main() {

	game := NewGame()

	ebiten.RunGame(game)

}

```

You also have the ability to use `File.OnLoop`, `File.OnFrameChange`, `File.OnTagEnter`, and `File.OnTagExit` callbacks to trigger events when an animation's state changes, for example. That's roughly it!

## Additional Notes

As for dependencies, GoAseprite makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 
