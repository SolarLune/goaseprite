package goaseprite

import (
	"bufio"
	"log"
	"os"

	"path/filepath"

	"strings"

	"strconv"

	"sort"

	"github.com/tidwall/gjson"
)

const (
	ASEPRITE_PLAY_FORWARD  = "forward"
	ASEPRITE_PLAY_BACKWARD = "reverse"
	ASEPRITE_PLAY_PINGPONG = "pingpong"
)

type Frame struct {
	X        int32
	Y        int32
	Duration float32
}

type Animation struct {
	Name      string
	Start     int32
	End       int32
	Direction string
}

type File struct {
	ImagePath        string
	FrameWidth       int32
	FrameHeight      int32
	Frames           []Frame
	Animations       []Animation
	CurrentAnimation *Animation
	frameCounter     float32
	CurrentFrame     int32
	prevCurrentFrame int32
	PrevFrame        int32
	PlaySpeed        float32
	Playing          bool
	pingpongedOnce   bool
}

func (this *File) Play(animName string) {
	cur := this.GetAnimation(animName)
	if cur == nil {
		log.Fatal(`Error: Animation named "` + animName + `" not found in Aseprite file!`)
	}
	if this.CurrentAnimation != cur {
		this.CurrentAnimation = cur
		this.CurrentFrame = this.CurrentAnimation.Start
		if this.CurrentAnimation.Direction == ASEPRITE_PLAY_BACKWARD {
			this.CurrentFrame = this.CurrentAnimation.End
		}
		this.pingpongedOnce = false
	}
}

func (this *File) Update(deltaTime float32) {

	this.PrevFrame = this.prevCurrentFrame
	this.prevCurrentFrame = this.CurrentFrame

	if this.CurrentAnimation != nil {

		this.frameCounter += deltaTime

		anim := this.CurrentAnimation

		if this.frameCounter > this.Frames[this.CurrentFrame].Duration {

			this.frameCounter = 0

			if anim.Direction == ASEPRITE_PLAY_FORWARD {
				this.CurrentFrame++
			} else if anim.Direction == ASEPRITE_PLAY_BACKWARD {
				this.CurrentFrame--
			} else if anim.Direction == ASEPRITE_PLAY_PINGPONG {
				if this.pingpongedOnce {
					this.CurrentFrame--
				} else {
					this.CurrentFrame++
				}
			}

		}

		if this.CurrentFrame > anim.End {
			if anim.Direction == ASEPRITE_PLAY_PINGPONG {
				this.pingpongedOnce = !this.pingpongedOnce
				this.CurrentFrame = anim.End - 1
				if this.CurrentFrame < anim.Start {
					this.CurrentFrame = anim.Start
				}
			} else {
				this.CurrentFrame = anim.Start
			}
		}

		if this.CurrentFrame < anim.Start {
			if anim.Direction == ASEPRITE_PLAY_PINGPONG {
				this.pingpongedOnce = !this.pingpongedOnce
				this.CurrentFrame = anim.Start + 1
				if this.CurrentFrame > anim.End {
					this.CurrentFrame = anim.End
				}

			} else {
				this.CurrentFrame = anim.End
			}
		}

	}

}

func (this *File) GetAnimation(animName string) *Animation {

	for index := range this.Animations {
		anim := &this.Animations[index]
		if anim.Name == animName {
			return anim
		}
	}

	return nil

}

func (this *File) GetFrameXY() (int32, int32) {

	var frameX, frameY int32 = -1, -1

	if this.CurrentAnimation != nil {

		frameX = this.Frames[this.CurrentFrame].X
		frameY = this.Frames[this.CurrentFrame].Y

	}

	return frameX, frameY

}

func (this *File) IsPlaying(animName string) bool {

	if this.CurrentAnimation != nil {
		if this.CurrentAnimation.Name == animName {
			return true
		}
	}

	return false
}

func (this *File) TouchingTag(tagName string) bool {
	for _, anim := range this.Animations {
		if anim.Name == tagName && this.CurrentFrame >= anim.Start && this.CurrentFrame <= anim.End {
			return true
		}
	}
	return false
}

func (this *File) TouchingTags() []string {
	anims := []string{}
	for _, anim := range this.Animations {
		if this.CurrentFrame >= anim.Start && this.CurrentFrame <= anim.End {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

func (this *File) HitTag(tagName string) bool {
	for _, anim := range this.Animations {
		if anim.Name == tagName && (this.CurrentFrame >= anim.Start && this.CurrentFrame <= anim.End) && (this.PrevFrame < anim.Start || this.PrevFrame > anim.End) {
			return true
		}
	}
	return false
}

func (this *File) HitTags() []string {
	anims := []string{}

	for _, anim := range this.Animations {
		if (this.CurrentFrame >= anim.Start && this.CurrentFrame <= anim.End) && (this.PrevFrame < anim.Start || this.PrevFrame > anim.End) {
			anims = append(anims, anim.Name)
		}
	}
	return anims
}

func (this *File) LeftTag(tagName string) bool {
	for _, anim := range this.Animations {
		if anim.Name == tagName && (this.PrevFrame >= anim.Start && this.PrevFrame <= anim.End) && (this.CurrentFrame < anim.Start || this.CurrentFrame > anim.End) {
			return true
		}
	}
	return false
}

func (this *File) LeftTags() []string {
	anims := []string{}

	for _, anim := range this.Animations {
		if (this.PrevFrame >= anim.Start && this.PrevFrame <= anim.End) && (this.CurrentFrame < anim.Start || this.CurrentFrame > anim.End) {
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

func ParseJson(filename string) File {

	file := readFile(filename)

	ase := File{}
	ase.Animations = make([]Animation, 0)

	wd, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	ase.ImagePath, err = filepath.Rel(wd, gjson.Get(file, "meta.image").String())

	if err != nil {
		log.Fatal(err)
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

	for _, anim := range gjson.Get(file, "meta.frameTags").Array() {

		ase.Animations = append(ase.Animations, Animation{
			Name:      anim.Get("name").Str,
			Start:     int32(anim.Get("from").Num),
			End:       int32(anim.Get("to").Num),
			Direction: anim.Get("direction").Str})

	}

	return ase
}
