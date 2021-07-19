// Package goaseprite is an Aseprite JSON loader written in Golang.
package goaseprite

import (
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

// Frame contains timing and position information for the frame on the spritesheet.
type Frame struct {
	X, Y     int
	Duration float32 // The duration of the frame in seconds.
}

// Slice represents a Slice (rectangle) that was defined in Aseprite and exported in the JSON file.
type Slice struct {
	Name  string      // Name is the name of the Slice, as specified in Aseprite.
	Data  string      // Data is blank by default, but can be specified on export from Aseprite to be whatever you need it to be.
	Keys  []*SliceKey // The individual keys (positions and sizes of Slices) according to the Frames they operate on.
	Color int64
}

// SliceKey represents a Slice's size and position in the Aseprite file. An individual Aseprite File can have multiple Slices inside,
// which can also have multiple frames in which the Slice's position and size changes. The SliceKey's Frame indicates which frame the key is
// operating on.
type SliceKey struct {
	Frame      int32
	X, Y, W, H int
}

// Tag contains details regarding each tag or animation from Aseprite.
// Start and End are the starting and ending frame of the Tag. Direction is a string, and can be assigned one of the playback constants.
type Tag struct {
	Name       string
	Start, End int
	Direction  string
	File       *File
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
	Path           string          // Path to the file (exampleSprite.json); blank if the *File was loaded using Read().
	ImagePath      string          // Path to the image associated with the Aseprite file (exampleSprite.png).
	Width, Height  int32           // Overall width and height of the File.
	Frames         []*Frame        // The animation Frames present in the File.
	Tags           map[string]*Tag // A map of Tags, with their names being the keys.
	Layers         []*Layer        // A slice of Layers.
	Slices         []*Slice        // A slice of the Slices present in the file.
	PlaySpeed      float32         // The playback speed; altering this can be used to globally slow down or speed up animation playback.
	CurrentTag     *Tag            // The currently playing animation.
	FrameIndex     int             // The current frame of the File's animation / tag playback.
	PrevFrameIndex int             // The previous frame in the playback.
	frameCounter   float32

	// Callbacks
	OnLoop        func()         // OnLoop gets called when the playing animation / tag does a complete loop. For a ping-pong animation, this is a full forward + back cycle.
	OnFrameChange func()         // OnFrameChange gets called when the playing animation / tag changes frames.
	OnTagEnter    func(tag *Tag) // OnTagEnter gets called when entering a tag from "outside" of it (i.e. if not playing a tag and then it gets played, this gets called, or if you're playing a tag and you pass through another tag).
	OnTagExit     func(tag *Tag) // OnTagExit gets called when exiting a tag from inside of it (i.e. if you finish passing through a tag while playing another one).

	playDirection int
}

// Play sets the specified tag name up to be played back.
func (file *File) Play(tagName string) {

	if anim, exists := file.Tags[tagName]; exists && anim != file.CurrentTag {

		if file.CurrentTag == nil {
			file.PrevFrameIndex = -1
		} else {
			file.PrevFrameIndex = file.FrameIndex
		}

		file.CurrentTag = anim
		file.frameCounter = 0

		if anim.Direction == PlayBackward {
			file.playDirection = -1
			file.FrameIndex = file.CurrentTag.End
		} else {
			file.playDirection = 1
			file.FrameIndex = file.CurrentTag.Start
		}

		file.pollTagChanges()

	}

}

// Update updates the currently playing animation. dt is the delta value between the previous frame and the current frame.
func (file *File) Update(dt float32) {

	anim := file.CurrentTag

	if anim != nil {

		file.frameCounter += dt * file.PlaySpeed

		frameDur := file.Frames[file.FrameIndex].Duration

		for file.frameCounter >= frameDur {

			file.frameCounter -= frameDur

			file.PrevFrameIndex = file.FrameIndex

			file.FrameIndex += file.playDirection

			if anim.Direction == PlayPingPong {

				if file.FrameIndex > anim.End {
					file.FrameIndex = anim.End - 1
					file.playDirection *= -1
				} else if file.FrameIndex < anim.Start {
					file.FrameIndex = anim.Start + 1
					file.playDirection *= -1
					if file.OnLoop != nil {
						file.OnLoop()
					}
				}

			} else if file.playDirection > 0 && file.FrameIndex > anim.End {
				file.FrameIndex -= anim.End - anim.Start + 1
				if file.OnLoop != nil {
					file.OnLoop()
				}
			} else if file.playDirection < 0 && file.FrameIndex < anim.Start {
				file.FrameIndex += anim.End - anim.Start + 1
				if file.OnLoop != nil {
					file.OnLoop()
				}
			}

			if file.FrameIndex != file.PrevFrameIndex && file.OnFrameChange != nil {
				file.OnFrameChange()
			}

			file.pollTagChanges()

		}

	}

}

// pollTagChanges polls the File for tag changes (entering or exiting Tags).
func (file *File) pollTagChanges() {

	if file.OnTagExit != nil {
		for _, tag := range file.Tags {
			if (file.PrevFrameIndex >= tag.Start && file.PrevFrameIndex <= tag.End) && (file.FrameIndex < tag.Start || file.FrameIndex > tag.End) {
				file.OnTagExit(tag)
			}
		}
	}

	if file.OnTagEnter != nil {
		for _, tag := range file.Tags {
			if (file.PrevFrameIndex < tag.Start || file.PrevFrameIndex > tag.End) && (file.FrameIndex >= tag.Start && file.FrameIndex <= tag.End) {
				file.OnTagEnter(tag)
			}
		}
	}

}

// CurrentFrame returns the current frame for the currently playing Tag in the File. Note that if a Tag isn't playing back, CurrentFrame() returns nil.
func (file *File) CurrentFrame() *Frame {
	if file.CurrentTag != nil {
		return file.Frames[file.FrameIndex]
	}
	return nil
}

// SetFrame sets the currently visible frame to frameIndex, using the playing animation as the range (so a frameIndex of 0 would set it to the first frame of an animation that is playing).
func (file *File) SetFrame(frameIndex int) {

	if file.CurrentTag != nil {

		file.FrameIndex = file.CurrentTag.Start + frameIndex
		if file.FrameIndex > file.CurrentTag.End {
			file.FrameIndex = file.CurrentTag.End
		}

	}

}

// SliceByName returns a Slice that has the name specified. Note that a File can have multiple Slices by the same name.
func (file *File) SliceByName(sliceName string) *Slice {
	for _, slice := range file.Slices {
		if slice.Name == sliceName {
			return slice
		}
	}
	return nil
}

// HasSlice returns true if the File has a Slice of the specified name.
func (file *File) HasSlice(sliceName string) bool {
	return file.SliceByName(sliceName) != nil
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
		Tags:      map[string]*Tag{},
		ImagePath: filepath.Clean(gjson.Get(json, "meta.image").String()),
		PlaySpeed: 1,
	}

	frameNames := []string{}

	for _, key := range gjson.Get(json, "meta.layers").Array() {
		ase.Layers = append(ase.Layers, &Layer{Name: key.Get("name").String(), Opacity: uint8(key.Get("opacity").Int()), BlendMode: key.Get("blendMode").String()})
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

		frame := &Frame{}
		frame.X = int(frameData.Get("frame.x").Num)
		frame.Y = int(frameData.Get("frame.y").Num)
		frame.Duration = float32(frameData.Get("duration").Num) / 1000

		ase.Frames = append(ase.Frames, frame)

		// We want to set it only on the first frame loaded
		if ase.Width == 0 {
			ase.Width = int32(frameData.Get("sourceSize.w").Num)
			ase.Height = int32(frameData.Get("sourceSize.h").Num)
		}

	}

	// Default ("") animation
	ase.Tags[""] = &Tag{
		Name:      "",
		Start:     0,
		End:       len(ase.Frames) - 1,
		Direction: PlayForward,
	}

	for _, anim := range gjson.Get(json, "meta.frameTags").Array() {

		animName := anim.Get("name").Str
		ase.Tags[animName] = &Tag{
			Name:      animName,
			Start:     int(anim.Get("from").Num),
			End:       int(anim.Get("to").Num),
			Direction: anim.Get("direction").Str,
		}

	}

	for _, sliceData := range gjson.Get(json, "meta.slices").Array() {

		color, _ := strconv.ParseInt("0x"+sliceData.Get("color").Str[1:], 0, 64)

		newSlice := &Slice{
			Name:  sliceData.Get("name").Str,
			Data:  sliceData.Get("data").Str,
			Color: color,
		}

		for _, sdKey := range sliceData.Get("keys").Array() {
			newSlice.Keys = append(newSlice.Keys, &SliceKey{
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
