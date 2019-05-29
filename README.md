# goaseprite

[GoDoc link](https://godoc.org/github.com/SolarLune/goaseprite)

Hello! This is a JSON loader for Aseprite files written in / for Go.

## How To Use

Usage is pretty straightforward. You export a sprite sheet from Aseprite (Ctrl+E), with Output File and JSON data checked, and the values set to Hash with Frame Tags and Slices (optionally) on.

Then you'll want to load the Aseprite data. To do this, you'll call `goaseprite.Open()` with a string argument of where to find the Aseprite JSON data file, or `goaseprite.ReadFile()` with the data itself in something that implements io.Reader. Either way, you'll get a `goaseprite.File`. It's from here that you control your animation.

After you have a File, you can just call the `File.Update()` function with an argument of delta time (the time between the previous frame and the current one) to get it updating. After that, use `File.Play()` to play an animation, and call `File.GetFrameXY()` to get the X and Y position of the current frame on the sprite sheet for where on the source sprite sheet to pull the frame of animation from. 

Here's a quick pseudo-example for a simple "Player" class using [raylib-go](https://github.com/gen2brain/raylib-go):

```go
package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/solarlune/GoAseprite"
)


type Player struct {
    Ase         *goaseprite.File
    Texture     rl.Texture2D
    TextureRect rl.Rectangle
}

func NewPlayer() *Player {

    player := Player{}
    
    // goaseprite.Open() returns an AsepriteFile, assuming it finds the JSON file. You can also use goaseprite.Read( io.Reader ) to read 
    // in JSON info from data.
    player.Ase = goaseprite.Open("assets/graphics/Player.json")
    
    // Note that while File.ImagePath exists, it will be the absolute path to the image file as exported from Aseprite, so it's best to load 
    // the texture yourself using relative paths so you can distribute it for others' computers.
    player.Texture = rl.LoadTexture("assets/graphics/Player.png")
    
    // Set up the texture rectangle for drawing the sprite.
    player.TextureRect = rl.Rectangle{0, 0, player.Ase.FrameWidth, player.Ase.FrameHeight}
    
    // Queues up the "Play" animation.
    player.Ase.Play("Idle")
    
    return &player
    
}

func (player *Player) Update() {
    
    // Call this every frame with the delta-time (time since the last frame); this will update the File's 
    // currently playing animation.
    player.Ase.Update(rl.GetFrameTime())
    
    // Set up the source rectangle for drawing the sprite (on the sprite sheet). File.GetFrameXY() will return the X and Y position
    // of the current frame of animation for the File.
    x, y := this.Ase.GetFrameXY()
    player.TextureRect.X = float32(x)
    player.TextureRect.Y = float32(y)
}

func (player *Player) Draw() {
    // Draw it!
    rl.DrawTextureRec(player.Texture, player.TextureRect, rl.Vector2{0, 0}, rl.White)
}

```

## Additional Notes

As for dependencies, GoAseprite makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 
