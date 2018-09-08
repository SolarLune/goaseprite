# goaseprite
Aseprite JSON loader for Go (Golang)

Yo, 'sup! This is a JSON loader for Aseprite files written in / for Go.

## How To Use

Usage is pretty straightforward. You export a sprite sheet from Aseprite (Ctrl+E), with a vertical or horizontal strip and an Output File, JSON data checked, and the values set to Hash with Frame Tags on (the default).

For use with your project, you'll call goaseprite.New() with a string argument of where to find the outputted Aseprite JSON data file. The function will return a goaseprite.AsepriteFile struct. It's from here that you control your animation. Since each instance of an AsepriteFile struct has playback specific variables and values, it's best not to share these across multiple instances (unless they are supposed to share animation playback).

After you have an AsepriteFile, you can just call its `Update()` function with an argument of delta time (the time between the previous frame and the current one) to get it updating, and call `GetFrameXY()` to get the X and Y position of the current frame on the sprite sheet. Here's a quick pseudo-example for a simple "Player" class using [raylib-go](https://github.com/gen2brain/raylib-go):

```go
package main

import (
	"github.com/gen2brain/raylib-go/raylib"
	ase "github.com/solarlune/GoAseprite"
)


type Player struct {
    Ase         ase.AsepriteFile
    Texture     raylib.Texture2D
    TextureRect raylib.Rectangle
}

func NewPlayer() *Player {

    player := Player{}
    
    // goaseprite.New() returns an AsepriteFile, assuming it finds the JSON file
    player.Ase = ase.New("assets/graphics/Player.json")
    
    // AsepriteFile.ImagePath will be relative to the working directory
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
    this.TextureRect.X, this.TextureRect.Y = this.Ase.GetFrameXY()
}

func (this *Player) Draw() {
    // And draw it~!
    raylib.DrawTextureRec(
    this.Texture,
    this.TextureRect,
    raylib.Vector2{0, 0},
    raylib.White)
}

```

Take a look at the Wiki for more information on the API!

## Additional Notes

As for dependencies, GoAseprite makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 
