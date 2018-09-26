package goaseprite

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	// AsepritePlayForward plays animations forward
	AsepritePlayForward = "forward"
	// AsepritePlayBackward plays animations backwards
	AsepritePlayBackward = "reverse"
	// AsepritePlayPingPong plays animation forward then backward
	AsepritePlayPingPong = "pingpong"
)

// AsepriteFrame contains the frame information
type AsepriteFrame struct {
	X        int32
	Y        int32
	Duration float32
}

// AsepriteAnimation contains the details of each tagged animation
type AsepriteAnimation struct {
	Name      string
	Start     int32
	End       int32
	Direction string
}

// AsepriteFile contains all properties of an exported aseprite file.
type AsepriteFile struct {
	ImagePath        string
	FrameWidth       int32
	FrameHeight      int32
	Frames           []AsepriteFrame
	Animations       []AsepriteAnimation
	CurrentAnimation *AsepriteAnimation
	frameCounter     float32
	CurrentFrame     int32
	prevCurrentFrame int32
	PrevFrame        int32
	PlaySpeed        float32
	Playing          bool
	pingpongedOnce   bool
}

// Play Queues playback of the specified animation (assuming it can be found).
func (asf *AsepriteFile) Play(animName string) {
	cur := asf.GetAnimation(animName)
	if cur == nil {
		log.Fatal(`Error: Animation named "` + animName + `" not found in Aseprite file!`)
	}
	if asf.CurrentAnimation != cur {
		asf.CurrentAnimation = cur
		asf.CurrentFrame = asf.CurrentAnimation.Start
		if asf.CurrentAnimation.Direction == AsepritePlayBackward {
			asf.CurrentFrame = asf.CurrentAnimation.End
		}
		asf.pingpongedOnce = false
	}
}

// Update Steps the file forward in time, updating the currently playing
// animation (and also handling looping).
func (asf *AsepriteFile) Update(deltaTime float32) {

	asf.PrevFrame = asf.prevCurrentFrame
	asf.prevCurrentFrame = asf.CurrentFrame

	if asf.CurrentAnimation != nil {

		asf.frameCounter += deltaTime * asf.PlaySpeed

		anim := asf.CurrentAnimation

		if asf.frameCounter > asf.Frames[asf.CurrentFrame].Duration {

			asf.frameCounter = 0

			if anim.Direction == AsepritePlayForward {
				asf.CurrentFrame++
			} else if anim.Direction == AsepritePlayBackward {
				asf.CurrentFrame--
			} else if anim.Direction == AsepritePlayPingPong {
				if asf.pingpongedOnce {
					asf.CurrentFrame--
				} else {
					asf.CurrentFrame++
				}
			}

		}

		if asf.CurrentFrame > anim.End {
			if anim.Direction == AsepritePlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.End - 1
				if asf.CurrentFrame < anim.Start {
					asf.CurrentFrame = anim.Start
				}
			} else {
				asf.CurrentFrame = anim.Start
			}
		}

		if asf.CurrentFrame < anim.Start {
			if anim.Direction == AsepritePlayPingPong {
				asf.pingpongedOnce = !asf.pingpongedOnce
				asf.CurrentFrame = anim.Start + 1
				if asf.CurrentFrame > anim.End {
					asf.CurrentFrame = anim.End
				}

			} else {
				asf.CurrentFrame = anim.End
			}
		}

	}

}

// GetAnimation Returns a pointer to an AsepriteAnimation of the desired name. If it can't
// be found, it will return `nil`.
func (asf *AsepriteFile) GetAnimation(animName string) *AsepriteAnimation {

	for index := range asf.Animations {
		anim := &asf.Animations[index]
		if anim.Name == animName {
			return anim
		}
	}

	return nil

}

// GetFrameXY Returns the current frame's X and Y coordinates on the source sprite sheet
// for drawing the sprite.
func (asf *AsepriteFile) GetFrameXY() (int32, int32) {

	var frameX, frameY int32 = -1, -1

	if asf.CurrentAnimation != nil {

		frameX = asf.Frames[asf.CurrentFrame].X
		frameY = asf.Frames[asf.CurrentFrame].Y

	}

	return frameX, frameY

}

// IsPlaying Returns if the named animation is playing.
func (asf *AsepriteFile) IsPlaying(animName string) bool {

	if asf.CurrentAnimation != nil {
		if asf.CurrentAnimation.Name == animName {
			return true
		}
	}

	return false
}

// TouchingTag Returns if the AsepriteFile's playback is touching a tag (animation) by
// the specified name.
func (asf *AsepriteFile) TouchingTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			return true
		}
	}
	return false
}

// TouchingTags Returns a list of tags the playback is touching.
func (asf *AsepriteFile) TouchingTags() []string {
	anims := []string{}
	for _, anim := range asf.Animations {
		if asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// HitTag Returns if the AsepriteFile's playback just touched a tag by the specified name.
func (asf *AsepriteFile) HitTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			return true
		}
	}
	return false
}

// HitTags Returns a list of tags the AsepriteFile just touched.
func (asf *AsepriteFile) HitTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.CurrentFrame >= anim.Start && asf.CurrentFrame <= anim.End) && (asf.PrevFrame < anim.Start || asf.PrevFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

// LeftTag Returns if the AsepriteFile's playback just left a tag by the specified name.
func (asf *AsepriteFile) LeftTag(tagName string) bool {
	for _, anim := range asf.Animations {
		if anim.Name == tagName && (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			return true
		}
	}
	return false
}

// LeftTags Returns a list of tags the AsepriteFile just left.
func (asf *AsepriteFile) LeftTags() []string {
	anims := []string{}

	for _, anim := range asf.Animations {
		if (asf.PrevFrame >= anim.Start && asf.PrevFrame <= anim.End) && (asf.CurrentFrame < anim.Start || asf.CurrentFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

func readFile(filePath string) string {

	file, err := os.Open(filePath)

	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)

	out := ""

	for scanner.Scan() {
		out += scanner.Text()
	}

	file.Close()

	return out

}

type byFrameNumber []string

func (b byFrameNumber) Len() int {
	return len(b)
}
func (b byFrameNumber) Swap(x, y int) {
	b[x], b[y] = b[y], b[x]
}
func (b byFrameNumber) Less(xi, yi int) bool {
	x := b[xi]
	y := b[yi]
	xfi := strings.LastIndex(x, " ") + 1
	xli := strings.LastIndex(x, ".")
	xv, _ := strconv.ParseInt(x[xfi:xli], 10, 32)
	yfi := strings.LastIndex(y, " ") + 1
	yli := strings.LastIndex(y, ".")
	yv, _ := strconv.ParseInt(y[yfi:yli], 10, 32)
	return xv < yv
}

// Load parses and returns an AsepriteFile. Your starting point.
func Load(aseJSONFilePath string) AsepriteFile {

	file := readFile(aseJSONFilePath)

	ase := AsepriteFile{}
	ase.Animations = make([]AsepriteAnimation, 0)
	ase.PlaySpeed = 1

	if path, err := filepath.Abs(gjson.Get(file, "meta.image").String()); err != nil {
		log.Fatalln(err)
	} else {
		ase.ImagePath = path
	}

	frameNames := []string{}

	for key := range gjson.Get(file, "frames").Map() {
		frameNames = append(frameNames, key)
	}

	sort.Sort(byFrameNumber(frameNames))

	for _, key := range frameNames {

		frameName := key
		frameName = strings.Replace(frameName, ".", `\.`, -1)
		frameData := gjson.Get(file, "frames."+frameName)

		frame := AsepriteFrame{}
		frame.X = int32(frameData.Get("frame.x").Num)
		frame.Y = int32(frameData.Get("frame.y").Num)
		frame.Duration = float32(frameData.Get("duration").Num) / 1000

		ase.Frames = append(ase.Frames, frame)

		if ase.FrameWidth == 0 {
			ase.FrameWidth = int32(frameData.Get("sourceSize.w").Num)
			ase.FrameHeight = int32(frameData.Get("sourceSize.h").Num)
		}

	}

	for _, anim := range gjson.Get(file, "meta.frameTags").Array() {

		ase.Animations = append(ase.Animations, AsepriteAnimation{
			Name:      anim.Get("name").Str,
			Start:     int32(anim.Get("from").Num),
			End:       int32(anim.Get("to").Num),
			Direction: anim.Get("direction").Str})

	}

	return ase
}
