package spritesheet

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"

	"github.com/faiface/pixel"
)

// Cache for storing sprites to avoid recreating them
var (
	spriteMutex  sync.RWMutex
	spriteCache  = make(map[int]pixel.Picture)
	pictureCache = make(map[string]pixel.Picture)
)

// LoadSpriteSheet takes a path to a resource and how it should be divided and returns
// a funciton to optain the sprite at that index
func LoadSpriteSheet(path string, row, col int) (func(int) pixel.Picture, error) {
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Load Image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// Check if tile is square
	b := img.Bounds()
	if b.Max.X/col != b.Max.Y/row {
		fmt.Println("width/col = ", b.Max.X, ", height/row = ", b.Max.Y)
		return nil, fmt.Errorf(fmt.Sprintf("Invalid dimensions (%d, %d) for sprite sheet %s\n", row, col, path))
	}

	tileSize := b.Max.X / col

	return func(i int) pixel.Picture {
		if i < 0 || i >= row*col {
			panic("Index out of bounds for sprite sheet")
		}

		// Check if this sprite is already in the cache
		spriteMutex.RLock()
		cachedSprite, exists := spriteCache[i]
		spriteMutex.RUnlock()

		if exists {
			return cachedSprite
		}

		// If not in cache, create it
		r := i / col
		c := i % col

		subImage := img.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(image.Rect(c*tileSize, r*tileSize, (c+1)*tileSize, (r+1)*tileSize))

		picData := pixel.PictureDataFromImage(subImage)

		// Store in cache for future use
		spriteMutex.Lock()
		spriteCache[i] = picData
		spriteMutex.Unlock()

		return picData
	}, nil
}

func LoadPicture(path string) (pixel.Picture, error) {
	// Check if the picture is already cached
	spriteMutex.RLock()
	cachedPic, exists := pictureCache[path]
	spriteMutex.RUnlock()

	if exists {
		return cachedPic, nil
	}

	// If not in cache, load it
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	pic := pixel.PictureDataFromImage(img)

	// Store in cache
	spriteMutex.Lock()
	pictureCache[path] = pic
	spriteMutex.Unlock()

	return pic, nil
}

// Background image caching
var (
	playBGPic      pixel.Picture
	nextPieceBGPic pixel.Picture
	playBGOnce     sync.Once
	nextPieceBGOnce sync.Once
)

func GetPlayBGPic() pixel.Picture {
	playBGOnce.Do(func() {
		blackImg := image.NewRGBA(image.Rect(0, 0, 200, 400))
		for x := 0; x < 200; x++ {
			for y := 0; y < 400; y++ {
				blackImg.SetRGBA(x, y, color.RGBA{0x00, 0x00, 0x00, 0xA0})
			}
		}
		playBGPic = pixel.PictureDataFromImage(blackImg)
	})
	return playBGPic
}

func GetNextPieceBGPic() pixel.Picture {
	nextPieceBGOnce.Do(func() {
		blackImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for x := 0; x < 100; x++ {
			for y := 0; y < 100; y++ {
				blackImg.SetRGBA(x, y, color.RGBA{0x00, 0x00, 0x00, 0xA0})
			}
		}
		nextPieceBGPic = pixel.PictureDataFromImage(blackImg)
	})
	return nextPieceBGPic
}
