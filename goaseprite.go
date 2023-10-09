// Package goaseprite is an Aseprite JSON loader written in Golang.
package goaseprite

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	PlayForward  = "forward"  // PlayForward plays animations forward
	PlayBackward = "reverse"  // PlayBackward plays animations backwards
	PlayPingPong = "pingpong" // PlayPingPong plays animation forward then backward
)

const (
	ErrorNoTagByName = "no tags by name"
)

// Frame contains timing and position information for the frame on the spritesheet.
type Frame struct {
	X, Y     int
	Duration float32 // The duration of the frame in seconds.
}

// Slice represents a Slice (rectangle) that was defined in Aseprite and exported in the JSON file.
type Slice struct {
	Name  string     // Name is the name of the Slice, as specified in Aseprite.
	Data  string     // Data is blank by default, but can be specified on export from Aseprite to be whatever you need it to be.
	Keys  []SliceKey // The individual keys (positions and sizes of Slices) according to the Frames they operate on.
	Color int64
}

func (slice Slice) IsEmpty() bool {
	return len(slice.Keys) == 0
}

// SliceKey represents a Slice's size and position in the Aseprite file on a specific frame. An individual Aseprite File can have multiple
// Slices inside, which can also have multiple frames in which the Slice's position and size changes. The SliceKey's Frame indicates which
// frame the key is operating on.
type SliceKey struct {
	Frame      int32
	X, Y, W, H int
}

// Center returns the center X and Y position of the Slice in the current key.
func (key SliceKey) Center() (int, int) {
	return key.X + (key.W / 2), key.Y + (key.H / 2)
}

// Tag contains details regarding each tag or animation from Aseprite.
// Start and End are the starting and ending frame of the Tag. Direction is a string, and can be assigned one of the playback constants.
type Tag struct {
	Name       string
	Start, End int
	Direction  string
	File       *File
}

func (tag Tag) IsEmpty() bool {
	return tag.File == nil
}

// Layer contains details regarding the layers exported from Aseprite, including the layer's name (string), opacity (0-255), and
// blend mode (string).
type Layer struct {
	Name      string
	Opacity   uint8
	BlendMode string
}

// File contains all properties of an exported aseprite file. ImagePath is the absolute path to the image as reported by the exported
// Aseprite JSON data. Path is the string used to open the File if it was opened with the Open() function; otherwise, it's blank.
type File struct {
	Path                    string  // Path to the file (exampleSprite.json); blank if the *File was loaded using Read().
	ImagePath               string  // Path to the image associated with the Aseprite file (exampleSprite.png).
	Width, Height           int32   // Overall width and height of the File.
	FrameWidth, FrameHeight int32   // Width and height of the frames in the File.
	Frames                  []Frame // The animation Frames present in the File.
	Tags                    []Tag   // A map of Tags, with their names being the keys.
	Layers                  []Layer // A slice of Layers.
	Slices                  []Slice // A slice of the Slices present in the file.
}

// SliceByName returns a Slice that has the name specified and a boolean indicating whether it could be found or not.
// Note that a File can have multiple Slices by the same name.
func (file *File) SliceByName(sliceName string) (Slice, bool) {
	for _, slice := range file.Slices {
		if slice.Name == sliceName {
			return slice, true
		}
	}
	return Slice{}, false
}

// HasSlice returns true if the File has a Slice of the specified name.
func (file *File) HasSlice(sliceName string) bool {
	_, exists := file.SliceByName(sliceName)
	return exists
}

// Player is an animation player for Aseprite files.
type Player struct {
	File           *File
	PlaySpeed      float32 // The playback speed; altering this can be used to globally slow down or speed up animation playback.
	CurrentTag     Tag     // The currently playing animation.
	FrameIndex     int     // The current frame of the File's animation / tag playback.
	PrevFrameIndex int     // The previous frame in the playback.
	frameCounter   float32

	updateRanFirst bool
	prevUVX        float64
	prevUVY        float64

	// Callbacks
	OnLoop        func()        // OnLoop gets called when the playing animation / tag does a complete loop. For a ping-pong animation, this is a full forward + back cycle.
	OnFrameChange func()        // OnFrameChange gets called when the playing animation / tag changes frames.
	OnTagEnter    func(tag Tag) // OnTagEnter gets called when entering a tag from "outside" of it (i.e. if not playing a tag and then it gets played, this gets called, or if you're playing a tag and you pass through another tag).
	OnTagExit     func(tag Tag) // OnTagExit gets called when exiting a tag from inside of it (i.e. if you finish passing through a tag while playing another one).

	playDirection int
}

// CreatePlayer returns a new animation player that plays animations from a given Aseprite file.
func (file *File) CreatePlayer() *Player {
	return &Player{
		File:      file,
		PlaySpeed: 1,
	}
}

// Clone clones the Player.
func (player *Player) Clone() *Player {
	newPlayer := player.File.CreatePlayer()
	newPlayer.PlaySpeed = player.PlaySpeed
	newPlayer.CurrentTag = player.CurrentTag
	newPlayer.FrameIndex = player.FrameIndex
	newPlayer.frameCounter = player.frameCounter

	newPlayer.OnLoop = player.OnLoop
	newPlayer.OnFrameChange = player.OnFrameChange
	newPlayer.OnTagEnter = player.OnTagEnter
	newPlayer.OnTagExit = player.OnTagExit

	return newPlayer
}

// Play sets the specified tag name up to be played back. A tagName of "" will play back the entire file.
func (player *Player) Play(tagName string) error {

	exists := false

	for _, anim := range player.File.Tags {

		if anim.Name == tagName {

			exists = true

			if anim != player.CurrentTag {

				if !player.CurrentTag.IsEmpty() {
					player.PrevFrameIndex = -1
				} else {
					player.PrevFrameIndex = player.FrameIndex
				}

				player.CurrentTag = anim
				player.frameCounter = 0

				if anim.Direction == PlayBackward {
					player.playDirection = -1
					player.FrameIndex = player.CurrentTag.End
				} else {
					player.playDirection = 1
					player.FrameIndex = player.CurrentTag.Start
				}

				player.pollTagChanges()

			}

			break

		}
	}

	if !exists {
		return errors.New(ErrorNoTagByName)
	}

	return nil

}

// Update updates the currently playing animation. dt is the delta value between the previous frame and the current frame.
func (player *Player) Update(dt float32) {

	anim := player.CurrentTag

	if !anim.IsEmpty() {

		player.frameCounter += dt * player.PlaySpeed

		frameDur := player.File.Frames[player.FrameIndex].Duration

		player.prevUVX, player.prevUVY = player.CurrentUVCoords()

		for player.frameCounter >= frameDur {

			player.frameCounter -= frameDur

			player.PrevFrameIndex = player.FrameIndex

			player.FrameIndex += player.playDirection

			if anim.Direction == PlayPingPong {

				if player.FrameIndex > anim.End {
					player.FrameIndex = anim.End - 1
					player.playDirection *= -1
				} else if player.FrameIndex < anim.Start {
					player.FrameIndex = anim.Start + 1
					player.playDirection *= -1
					if player.OnLoop != nil {
						player.OnLoop()
					}
				}

			} else if player.playDirection > 0 && player.FrameIndex > anim.End {
				player.FrameIndex -= anim.End - anim.Start + 1
				if player.OnLoop != nil {
					player.OnLoop()
				}
			} else if player.playDirection < 0 && player.FrameIndex < anim.Start {
				player.FrameIndex += anim.End - anim.Start + 1
				if player.OnLoop != nil {
					player.OnLoop()
				}
			}

			if player.FrameIndex != player.PrevFrameIndex && player.OnFrameChange != nil {
				player.OnFrameChange()
			}

			player.pollTagChanges()

		}

	}

}

// TouchingTags returns the tags currently being touched by the Player (tag).
func (player *Player) TouchingTags() []Tag {
	tags := []Tag{}
	for _, t := range player.File.Tags {
		if player.FrameIndex >= t.Start && player.FrameIndex <= t.End {
			tags = append(tags, t)
		}
	}
	return tags
}

// TouchingTagByName returns if a tag by the given name is being touched by the Player (tag).
func (player *Player) TouchingTagByName(tagName string) bool {
	for _, t := range player.File.Tags {
		if t.Name == tagName && player.FrameIndex >= t.Start && player.FrameIndex <= t.End {
			return true
		}
	}
	return false
}

// pollTagChanges polls the File for tag changes (entering or exiting Tags).
func (player *Player) pollTagChanges() {

	if player.OnTagExit != nil {
		for _, tag := range player.File.Tags {
			if (player.PrevFrameIndex >= tag.Start && player.PrevFrameIndex <= tag.End) && (player.FrameIndex < tag.Start || player.FrameIndex > tag.End) {
				player.OnTagExit(tag)
			}
		}
	}

	if player.OnTagEnter != nil {
		for _, tag := range player.File.Tags {
			if (player.PrevFrameIndex < tag.Start || player.PrevFrameIndex > tag.End) && (player.FrameIndex >= tag.Start && player.FrameIndex <= tag.End) {
				player.OnTagEnter(tag)
			}
		}
	}

}

// CurrentFrame returns the current frame for the currently playing Tag in the File and a boolean indicating if the Player is playing a Tag or not.
func (player *Player) CurrentFrame() (Frame, bool) {
	if !player.CurrentTag.IsEmpty() {
		return player.File.Frames[player.FrameIndex], true
	}
	return Frame{}, false
}

// CurrentFrameCoords returns the four corners of the current frame, of format (x1, y1, x2, y2). If File.CurrentFrame() is nil, it will instead
// return all -1's.
func (player *Player) CurrentFrameCoords() (int, int, int, int) {

	if frame, ok := player.CurrentFrame(); ok {
		return frame.X, frame.Y, frame.X + int(player.File.FrameWidth), frame.Y + int(player.File.FrameHeight)
	}

	return -1, -1, -1, -1

}

// CurrentUVCoords returns the top-left corner of the current frame, of format (x, y). If File.CurrentFrame() is nil, it will instead
// return (-1, -1).
func (player *Player) CurrentUVCoords() (float64, float64) {

	if frame, ok := player.CurrentFrame(); ok {
		return float64(frame.X) / float64(player.File.Width), float64(frame.Y) / float64(player.File.Height)
	}

	return -1, -1

}

// CurrentUVCoordsDelta returns the current UV Coords as a coordinate movement delta.
// For example, if an animation were to return the X-axis UV coordinates of :
// [ 0, 0, 0, 0, 0.5, 0.5, 0.5, 0.5 ],
// CurrentUVCoordsDelta would return [ 0, 0, 0, 0, 0.5, 0, 0, 0 ], as the UV coordinate
// only changes on that one frame in the middle, from 0 to 0.5. Once it goes to the end,
// it would return -0.5 to return back to the starting frame.
func (player *Player) CurrentUVCoordsDelta() (float64, float64) {
	currentX, currentY := player.CurrentUVCoords()
	return currentX - player.prevUVX, currentY - player.prevUVY
}

// SetFrameIndex sets the currently visible frame to frameIndex, using the playing animation as the range.
// This means calling SetFrameIndex with a frameIndex of 2 would set it to the third frame of the animation that is currently playing.
func (player *Player) SetFrameIndex(frameIndex int) {

	if !player.CurrentTag.IsEmpty() {

		player.FrameIndex = player.CurrentTag.Start + frameIndex
		if player.FrameIndex > player.CurrentTag.End {
			player.FrameIndex = player.CurrentTag.End
		}
		player.frameCounter = 0

	}

}

// FrameIndexInAnimation returns the currently visible frame index, using the playing animation as the range.
// This means that a FrameIndexInAnimation of 0 would be the first frame in the currently playing animation,
// regardless of what frame in the sprite strip that is).
// If no animation is being played, this function will return -1.
func (player *Player) FrameIndexInAnimation() int {
	if !player.CurrentTag.IsEmpty() {
		return player.FrameIndex - player.CurrentTag.Start
	}
	return -1
}

// Open will use os.ReadFile() to open the Aseprite JSON file path specified to parse the data. Returns a *goaseprite.File.
// This can be your starting point. Files created with Open() will put the JSON filepath used in the Path field.
func Open(jsonPath string) *File {

	fileData, err := os.ReadFile(jsonPath)

	if err != nil {
		log.Println(err)
	}

	asf := Read(fileData)
	asf.Path = jsonPath
	return asf

}

// Read returns a *goaseprite.File for a given sequence of bytes read from an Aseprite JSON file.
func Read(fileData []byte) *File {

	json := string(fileData)

	ase := &File{
		Tags:      []Tag{},
		ImagePath: filepath.Clean(gjson.Get(json, "meta.image").String()),
	}

	frameNames := []string{}

	ase.Width = int32(gjson.Get(json, "meta.size.w").Num)
	ase.Height = int32(gjson.Get(json, "meta.size.h").Num)

	for _, key := range gjson.Get(json, "meta.layers").Array() {
		ase.Layers = append(ase.Layers, Layer{Name: key.Get("name").String(), Opacity: uint8(key.Get("opacity").Int()), BlendMode: key.Get("blendMode").String()})
	}

	for key := range gjson.Get(json, "frames").Map() {
		frameNames = append(frameNames, key)
	}

	sort.Slice(frameNames, func(i, j int) bool {
		x := frameNames[i]
		y := frameNames[j]
		xfi := strings.LastIndex(x, " ") + 1
		xli := strings.LastIndex(x, ".")
		xv, _ := strconv.ParseInt(x[xfi:xli], 10, 32)
		yfi := strings.LastIndex(y, " ") + 1
		yli := strings.LastIndex(y, ".")
		yv, _ := strconv.ParseInt(y[yfi:yli], 10, 32)
		return xv < yv
	})

	for _, key := range frameNames {

		frameName := key
		frameName = strings.Replace(frameName, ".", `\.`, -1)
		frameData := gjson.Get(json, "frames."+frameName)

		frame := Frame{}
		frame.X = int(frameData.Get("frame.x").Num)
		frame.Y = int(frameData.Get("frame.y").Num)
		frame.Duration = float32(frameData.Get("duration").Num) / 1000

		ase.Frames = append(ase.Frames, frame)

		// We want to set it only on the first frame loaded
		if ase.FrameWidth == 0 {
			ase.FrameWidth = int32(frameData.Get("sourceSize.w").Num)
			ase.FrameHeight = int32(frameData.Get("sourceSize.h").Num)
		}

	}

	// Default ("") animation
	ase.Tags = append(ase.Tags, Tag{
		Name:      "",
		Start:     0,
		End:       len(ase.Frames) - 1,
		Direction: PlayForward,
		File:      ase,
	})

	for _, anim := range gjson.Get(json, "meta.frameTags").Array() {

		animName := anim.Get("name").Str
		ase.Tags = append(ase.Tags, Tag{
			Name:      animName,
			Start:     int(anim.Get("from").Num),
			End:       int(anim.Get("to").Num),
			Direction: anim.Get("direction").Str,
			File:      ase,
		})

	}

	for _, sliceData := range gjson.Get(json, "meta.slices").Array() {

		color, _ := strconv.ParseInt("0x"+sliceData.Get("color").Str[1:], 0, 64)

		newSlice := Slice{
			Name:  sliceData.Get("name").Str,
			Data:  sliceData.Get("data").Str,
			Color: color,
		}

		for _, sdKey := range sliceData.Get("keys").Array() {
			newSlice.Keys = append(newSlice.Keys, SliceKey{
				Frame: int32(sdKey.Get("frame").Int()),
				X:     int(sdKey.Get("bounds.x").Int()),
				Y:     int(sdKey.Get("bounds.y").Int()),
				W:     int(sdKey.Get("bounds.w").Int()),
				H:     int(sdKey.Get("bounds.h").Int()),
			})
		}

		ase.Slices = append(ase.Slices, newSlice)
	}

	return ase

}
