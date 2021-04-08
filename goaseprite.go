// Package goaseprite is an Aseprite JSON loader written in Golang.
package goaseprite

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// PlayForward plays animations forward
	PlayForward = "forward"
	// PlayBackward plays animations backwards
	PlayBackward = "reverse"
	// PlayPingPong plays animation forward then backward
	PlayPingPong = "pingpong"
)

// Frame contains timing and position information for the frame on the spritesheet. Note that Duration is in seconds.
type Frame struct {
	X        int32
	Y        int32
	Duration float32
}

// Slice represents a Slice (rectangle) that was defined in Aseprite and exported in the JSON file. Data by default is blank,
// but can be specified on export from Aseprite to be whatever you need it to be.
type Slice struct {
	Name  string
	Data  string
	Color int64
	Keys  []*SliceKey
}

// SliceKey represents a Slice's size and position in the Aseprite file. An individual Aseprite File can have multiple Slices inside,
// which can also have multiple frames in which the Slice's position and size changes. The SliceKey's Frame indicates which frame the key is
// operating on.
type SliceKey struct {
	Frame      int32
	X, Y, W, H int32
}

// Animation contains details regarding each animation from Aseprite. This also represents a tag in Aseprite and in goaseprite.
// Start and End are the starting and ending frame of the Animation. Direction is a string, and can be assigned one of the playback constants.
type Animation struct {
	Name      string
	Start     int32
	End       int32
	Direction string
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
	ImagePath        string
	Path             string
	FrameWidth       int32
	FrameHeight      int32
	Frames           []Frame
	Animations       []Animation
	Layers           []Layer
	CurrentAnimation *Animation
	frameCounter     float32
	CurrentFrame     int32
	prevCurrentFrame int32
	PrevFrame        int32
	PlaySpeed        float32
	Playing          bool
	Slices           []*Slice
	pingpongedOnce   bool
	// FinishedAnimation returns true if the animation is finished playing. When playing forward or backward, it returns true on the
	// frame that the File loops the animation (assuming the File gets Update() called every game frame). When playing using ping-pong,
	// this function will return true when a full loop is finished (when the File plays forwards and then backwards, and loops again).
	FinishedAnimation bool
}

// Play queues playback of the specified animation (assuming it's in the File).
func (asf *File) Play(animName string) {
	cur := asf.GetAnimation(animName)
	if cur == nil {
		log.Println(`Error: Animation named "` + animName + `" not found in Aseprite file!`)
	}
	if asf.CurrentAnimation != cur {
		asf.CurrentAnimation = cur
		asf.FinishedAnimation = false
		asf.CurrentFrame = asf.CurrentAnimation.Start
		if asf.CurrentAnimation.Direction == PlayBackward {
			asf.CurrentFrame = asf.CurrentAnimation.End
		}
		asf.pingpongedOnce = false
	}
}

// Update steps the File forward in time, updating the currently playing animation (and also handling looping).
func (asf *File) Update(deltaTime float32) {

	asf.PrevFrame = asf.prevCurrentFrame
	asf.prevCurrentFrame = asf.CurrentFrame

	asf.FinishedAnimation = false

	if asf.CurrentAnimation != nil {

		asf.frameCounter += deltaTime * asf.PlaySpeed

		anim := asf.CurrentAnimation

		if asf.frameCounter > asf.Frames[asf.CurrentFrame].Duration {

			asf.frameCounter = 0

			if anim.Direction == PlayForward {
				asf.CurrentFrame++
			} else if anim.Direction == PlayBackward {
				asf.CurrentFrame--
			} else if anim.Direction == PlayPingPong {
				if asf.pingpongedOnce {
					asf.CurrentFrame--
				} else {
					asf.CurrentFrame++
				}
			}

		}

		if asf.CurrentFrame > anim.End {
			if anim.Direction == PlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.End - 1
				if asf.CurrentFrame < anim.Start {
					asf.CurrentFrame = anim.Start
				}
			} else {
				asf.CurrentFrame = anim.Start
				asf.FinishedAnimation = true
			}
		}

		if asf.CurrentFrame < anim.Start {
			if anim.Direction == PlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.Start + 1
				asf.FinishedAnimation = true

				if asf.CurrentFrame > anim.End {
					asf.CurrentFrame = anim.End
				}

			} else {
				asf.CurrentFrame = anim.End
				asf.FinishedAnimation = true
			}
		}

	}

}

// GetAnimation returns a pointer to an Animation of the desired name. If it can't be found, it will return nil.
func (asf *File) GetAnimation(animName string) *Animation {

	for index := range asf.Animations {
		anim := &asf.Animations[index]
		if anim.Name == animName {
			return anim
		}
	}

	return nil

}

// HasAnimation returns true if the File has an Animation of the specified name.
func (asf *File) HasAnimation(animName string) bool {
	return asf.GetAnimation(animName) != nil
}

// GetSlice returns a Slice that has the name specified. Note that a File can have multiple Slices by the same name.
func (asf *File) GetSlice(sliceName string) *Slice {
	for _, slice := range asf.Slices {
		if slice.Name == sliceName {
			return slice
		}
	}
	return nil
}

// HasSlice returns true if the File has a Slice of the specified name.
func (asf *File) HasSlice(sliceName string) bool {
	return asf.GetSlice(sliceName) != nil
}

// GetFrameXY returns the current frame's X and Y coordinates on the source sprite sheet for drawing the sprite.
func (asf *File) GetFrameXY() (int32, int32) {

	var frameX, frameY int32 = 0, 0

	if asf.CurrentAnimation != nil {

		frameX = asf.Frames[asf.CurrentFrame].X
		frameY = asf.Frames[asf.CurrentFrame].Y

	}

	return frameX, frameY

}

// IsPlaying returns if the named animation is playing.
func (asf *File) IsPlaying(animName string) bool {

	if asf.CurrentAnimation != nil && asf.CurrentAnimation.Name == animName {
		return true
	}

	return false
}

// TouchingTag returns if the File's playback is touching a tag by the specified name.
func (asf *File) TouchingTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			return true
		}
	}
	return false
}

// TouchingTags returns a list of tags the playback is currently touching.
func (asf *File) TouchingTags() []string {
	anims := []string{}
	for _, anim := range asf.Animations {
		if asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// HitTag returns if the File's playback just touched a tag by the specified name.
func (asf *File) HitTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			return true
		}
	}
	return false
}

// HitTags returns a list of tags the File just touched by the file's playback on this frame.
func (asf *File) HitTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// LeftTag returns if the File's playback just left a tag by the specified name.
func (asf *File) LeftTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			return true
		}
	}
	return false
}

// LeftTags returns a list of tags the File's playback just passed through on the last frame.
func (asf *File) LeftTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// ReadFile will use os.ReadFile() to open the Aseprite JSON file path specified to return a *goaseprite.File, relative to the current
// working directory. This can be your starting point. Files created with ReadFile() will put the JSON filepath used in
// the returned File's Path field.
func ReadFile(jsonPath string) *File {

	file, err := os.Open(jsonPath)

	if err != nil {
		log.Println(err)
	}

	asf := ReadBytes(file)
	asf.Path = jsonPath
	return asf

}

// ReadBytes returns a *goaseprite.File for a given file handle, in case you are opening the file yourself from another
// source (i.e. using the mobile asset package). This can also be your starting point,
func ReadBytes(file io.Reader) *File {

	scanner := bufio.NewScanner(file)

	json := ""

	for scanner.Scan() {
		json += scanner.Text()
	}

	ase := &File{}
	ase.Animations = make([]Animation, 0)
	ase.PlaySpeed = 1
	ase.ImagePath = filepath.Clean(gjson.Get(json, "meta.image").String())

	frameNames := []string{}

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
		frame.X = int32(frameData.Get("frame.x").Num)
		frame.Y = int32(frameData.Get("frame.y").Num)
		frame.Duration = float32(frameData.Get("duration").Num) / 1000

		ase.Frames = append(ase.Frames, frame)

		if ase.FrameWidth == 0 {
			ase.FrameWidth = int32(frameData.Get("sourceSize.w").Num)
			ase.FrameHeight = int32(frameData.Get("sourceSize.h").Num)
		}

	}

	for _, anim := range gjson.Get(json, "meta.frameTags").Array() {

		ase.Animations = append(ase.Animations, Animation{
			Name:      anim.Get("name").Str,
			Start:     int32(anim.Get("from").Num),
			End:       int32(anim.Get("to").Num),
			Direction: anim.Get("direction").Str})

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
				X:     int32(sdKey.Get("bounds.x").Int()),
				Y:     int32(sdKey.Get("bounds.y").Int()),
				W:     int32(sdKey.Get("bounds.w").Int()),
				H:     int32(sdKey.Get("bounds.h").Int()),
			})
		}

		ase.Slices = append(ase.Slices, newSlice)
	}

	return ase

}
