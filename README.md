# GoAseprite
Aseprite JSON loader for Go (Golang)

Yo, 'sup! This is a JSON loader for Aseprite files written in / for Go.

# Requirements

Go. GoAseprite also makes use of tidwall's nice [gjson](https://github.com/tidwall/gjson) package. 

# How To Use

Usage is pretty straightforward. You export a sprite sheet from Aseprite (Ctrl+E), with a vertical or horizontal strip and an Output File, JSON data checked, and the values set to Hash with Frame Tags on.

For using with your project, you'll call goaseprite.ParseJson() with the argument of where to find the outputted Aseprite JSON data file. The function will return a goaseprite.AsepriteFile struct. The File struct represents "home" for an animation - it's from here that you control animation playback and check tags. Since each instance of a File has playback specific variables and values, it's best not to share these across multiple instances (unless they are supposed to share animations).

After you have a File, you can just call its `Update()` function with an argument of delta time to get it updating. Here's a quick pseudo-example (pretty close to being correct) using [raylib-go](https://github.com/gen2brain/raylib-go):

```go
package main

type Player struct {
    Ase         goaseprite.AsepriteFile
    Texture     raylib.Texture
    TextureRect raylib.Rectangle
}

func NewPlayer() *Player {

    player := Player{}
    
    // ParseJson returns an goaseprite.File, assuming it finds the JSON file
    player.Ase = goaseprite.ParseJson("assets/graphics/Player.json")
    
    // File.ImagePath will be relative to the JSON file
    player.Texture = raylib.LoadTexture(player.Ase.ImagePath)
    
    // Set up the texture rectangle for drawing the sprite
    player.TextureRect = raylib.Rectangle{0, 0, player.Ase.FrameWidth, player.Ase.FrameHeight}
    
    // Sets up the animation file's Play animation to play
    player.Ase.Play("Idle")
    
    return &player
    
}

func (this *Player) Update() {
    
    // Call this every frame with the delta-time
    this.ase.Update(raylib.GetFrameTime())
    
    // Set up the source rectangle for drawing the sprite (on the sprite sheet)
    this.TextureRect.X, this.TextureRect.Y = this.ase.GetFrameXY()
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

Take a look at the Wiki for more information on the API.