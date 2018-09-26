# goaseprite
Aseprite JSON loader for Go (Golang)

Yo, 'sup! This is a JSON loader for Aseprite files written in / for Go.

## How To Use

Usage is pretty straightforward. You export a sprite sheet from Aseprite (Ctrl+E), with Output File and JSON data checked, and the values set to Hash with Frame Tags and Slices (optionally) on.

For use with your project, you'll call goaseprite.Load() with a string argument of where to find the outputted Aseprite JSON data file. The function will return a goaseprite.File struct. It's from here that you control your animation. Since each instance of an File struct has playback specific variables and values, it's best not to share these across multiple instances (unless they are supposed to share animation playback).

After you have a File, you can just call its `File.Update()` function with an argument of delta time (the time between the previous frame and the current one) to get it updating. After that, use `File.Play()` to play a specific animation, and call `File.GetFrameXY()` to get the X and Y position of the current frame on the sprite sheet. Here's a quick pseudo-example for a simple "Player" class using [raylib-go](https://github.com/gen2brain/raylib-go):

```go
package main

import (
	"github.com/gen2brain/raylib-go/raylib"
	ase "github.com/solarlune/GoAseprite"
)


type Player struct {
    Ase         ase.File
    Texture     raylib.Texture2D
    TextureRect raylib.Rectangle
}

func NewPlayer() *Player {

    player := Player{}
    
    // goaseprite.Load() returns an AsepriteFile, assuming it finds the JSON file
    player.Ase = ase.Load("assets/graphics/Player.json")
    
    // AsepriteFile.ImagePath will be the absolute path to the image file.
    player.Texture = raylib.LoadTexture(player.Ase.ImagePath)
    
    // Set up the texture rectangle for drawing the sprite
    player.TextureRect = raylib.Rectangle{0, 0, player.Ase.FrameWidth, player.Ase.FrameHeight}
    
    // Queues up the "Play" animation
    player.Ase.Play("Idle")
    
    return &player
    
}

func (this *Player) Update() {
    
    // Call this every frame with the delta-time (time since the last frame)
    this.Ase.Update(raylib.GetFrameTime())
    
    // Set up the source rectangle for drawing the sprite (on the sprite sheet)
    x, y := this.Ase.GetFrameXY()
    this.TextureRect.X = float32(x)
    this.TextureRect.Y = float32(y)
}

func (this *Player) Draw() {
    // And draw it~!
    raylib.DrawTextureRec(this.Texture, this.TextureRect, raylib.Vector2{0, 0}, raylib.White)
}

```

Take a look at the Wiki for more information on the API!

## Additional Notes

As for dependencies, GoAseprite makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 
