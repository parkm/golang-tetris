package main

import (
	"fmt"
	_ "image/png"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"

	ss "github.com/zkry/golang-tetris/spritesheet"
)

// BoardRows is the height of the game board in terms of blocks
const BoardRows = 22

// BoardCols is the width of the game board in terms of blocks
const BoardCols = 10

// Point represents a coordinate on the game board with Point{row:0, col:0}
// representing the bottom left
type Point struct {
	row int
	col int
}

// Board is an array containing the entire game board pieces.
type Board [22][10]Block

// Block represents the color of the block
type Block int

// Different values a point on the grid can hold
const (
	Empty Block = iota
	Goluboy
	Siniy
	Pink
	Purple
	Red
	Yellow
	Green
	Gray
	GoluboySpecial
	SiniySpecial
	PinkSpecial
	PurpleSpecial
	RedSpecial
	YellowSpecial
	GreenSpecial
	GraySpecial
)

// Piece is a constant for a shape of piece. There are 7 classic pieces like L, and O
type Piece int

// Various values that the pieces can be
const (
	IPiece Piece = iota
	JPiece
	LPiece
	OPiece
	SPiece
	TPiece
	ZPiece
	NoPiece Piece = -1
)

// Shape is a type containing four points, which represents the four points
// making a contiguous 'piece'.
type Shape [4]Point

const levelLength = 60.0 // Time it takes for game speed up
const speedUpRate = 0.1  // Every new level, the amount the game speeds up by

// DAS (Delayed Auto Shift) and ARR (Auto Repeat Rate) constants
const (
	DASDelay = 0.17  // Delay before auto-shifting starts (in seconds)
	ARRRate  = 0.033 // Time between shifts once auto-shift starts (in seconds)
)

var gameBoard Board
var activeShape Shape // The shape that the player controls
var currentPiece Piece
var gravityTimer float64
var baseSpeed float64 = 0.8
var gravitySpeed float64 = 0.8
var lockDelay float64 = 0.5 // Time before piece locks when on ground
var lockDelayTimer float64 = 0
var lockResets int = 0      // Count lock delay resets (limit to 15)
var levelUpTimer float64 = levelLength
var gameOver bool = false
var leftRightTimer float64      // Timer for left/right movement DAS
var ARRTimer float64            // Timer for ARR
var lastMoveDirection int = 0   // Last direction moved (-1 left, 1 right)
var score int
var nextPiece Piece
var holdPiece Piece = NoPiece   // Piece being held
var canHold bool = true         // Whether player can hold a piece
var rotationState int = 0       // Current rotation state (0-3)
var pieceBag []Piece = nil      // 7-bag system for randomization
var lastMovementWasRotation bool = false // Used for T-spin detection
var lastRotationPoint Shape      // The shape before last rotation (for T-spin detection)

var blockGen func(int) pixel.Picture
var bgImgSprite pixel.Sprite
var gameBGSprite pixel.Sprite
var nextPieceBGSprite pixel.Sprite
var holdPieceBGSprite pixel.Sprite

func main() {
	pixelgl.Run(run)
}

// run is the main code for the game. Allows pixelgl to run on main thread
func run() {
	// Initialize the window
	windowWidth := 765.0
	windowHeight := 450.0
	cfg := pixelgl.WindowConfig{
		Title:  "Blockfall",
		Bounds: pixel.R(0, 0, windowWidth, windowHeight),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// Load Various Resources:
	// Matriax on opengameart.org
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	blockGen, err = ss.LoadSpriteSheet(pwd+"/resources/blocks.png", 2, 8)
	if err != nil {
		panic(err)
	}

	// Background image, by ansimuz on opengameart.org
	bgPic, err := ss.LoadPicture(pwd + "/resources/parallax-mountain-bg.png")
	if err != nil {
		panic(err)
	}
	bgImgSprite = *pixel.NewSprite(bgPic, bgPic.Bounds())

	// Game Background
	blackPic := ss.GetPlayBGPic()
	gameBGSprite = *pixel.NewSprite(blackPic, blackPic.Bounds())

	// Next Piece BG
	nextPiecePic := ss.GetNextPieceBGPic()
	nextPieceBGSprite = *pixel.NewSprite(nextPiecePic, nextPiecePic.Bounds())

	// Hold Piece BG (using same sprite as next piece)
	holdPieceBGSprite = *pixel.NewSprite(nextPiecePic, nextPiecePic.Bounds())

	// Initialize the 7-bag
	initializeBag()

	nextPiece = getNextPiece()
	gameBoard.addPiece() // Add initial Piece to game
	last := time.Now()
	for !win.Closed() && !gameOver {
		// Perform time processing events
		dt := time.Since(last).Seconds()
		last = time.Now()
		gravityTimer += dt
		levelUpTimer -= dt

		// Update lock delay timer if piece is on ground
		if gameBoard.isTouchingFloor() {
			lockDelayTimer += dt
			if lockDelayTimer >= lockDelay {
				gameBoard.lockPiece()
				lockDelayTimer = 0
				lockResets = 0
			}
		} else {
			lockDelayTimer = 0
		}

		// Time Functions:
		// Gravity
		if gravityTimer > gravitySpeed {
			gravityTimer -= gravitySpeed
			didCollide := gameBoard.applyGravity()
			if didCollide {
				score += 10
			}
		}

		// Speed up
		if levelUpTimer <= 0 {
			if baseSpeed > 0.2 {
				baseSpeed = math.Max(baseSpeed-speedUpRate, 0.2)
			}
			levelUpTimer = levelLength
			gravitySpeed = baseSpeed
		}

		// DAS and ARR movement implementation
		direction := 0
		if win.Pressed(pixelgl.KeyRight) {
			direction = 1
		} else if win.Pressed(pixelgl.KeyLeft) {
			direction = -1
		} else {
			leftRightTimer = 0
			ARRTimer = 0
			lastMoveDirection = 0
		}

		// Handle movement with DAS/ARR
		if direction != 0 {
			if direction != lastMoveDirection || leftRightTimer == 0 {
				// Initial tap or direction change
				gameBoard.movePiece(direction)
				leftRightTimer = DASDelay
				lastMoveDirection = direction

				// Reset lock delay if moved and on ground (up to 15 times)
				if gameBoard.isTouchingFloor() && lockResets < 15 {
					lockDelayTimer = 0
					lockResets++
				}
			} else {
				// Auto-shift handling
				leftRightTimer -= dt
				if leftRightTimer <= 0 {
					// DAS has been charged, use ARR
					ARRTimer += dt
					if ARRTimer >= ARRRate {
						gameBoard.movePiece(direction)
						ARRTimer -= ARRRate

						// Reset lock delay if moved and on ground (up to 15 times)
						if gameBoard.isTouchingFloor() && lockResets < 15 {
							lockDelayTimer = 0
							lockResets++
						}
					}
				}
			}
		}

		if win.JustPressed(pixelgl.KeyDown) {
			gravitySpeed = 0.05
			if gravityTimer > 0.05 {
				gravityTimer = 0.05
			}
		}
		if win.JustReleased(pixelgl.KeyDown) {
			gravitySpeed = baseSpeed
		}
		if win.JustPressed(pixelgl.KeyUp) {
			gameBoard.rotatePiece(1) // Clockwise rotation

			// Reset lock delay if rotated and on ground (up to 15 times)
			if gameBoard.isTouchingFloor() && lockResets < 15 {
				lockDelayTimer = 0
				lockResets++
			}
		}
		if win.JustPressed(pixelgl.KeyZ) {
			gameBoard.rotatePiece(-1) // Counter-clockwise rotation

			// Reset lock delay if rotated and on ground (up to 15 times)
			if gameBoard.isTouchingFloor() && lockResets < 15 {
				lockDelayTimer = 0
				lockResets++
			}
		}
		if win.JustPressed(pixelgl.KeySpace) {
			gameBoard.instafall()
			score += 12
		}
		if win.JustPressed(pixelgl.KeyC) && canHold {
			gameBoard.holdPiece()
		}

		// Display Functions
		win.Clear(colornames.Black)
		displayBG(win)
		displayText(win)
		displayHoldPiece(win)
		gameBoard.displayBoard(win)
		win.Update()
	}
}

func displayText(win *pixelgl.Window) {
	// Text Generator
	scoreTextLocX := 500.0
	scoreTextLocY := 400.0
	basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	scoreTxt := text.New(pixel.V(scoreTextLocX, scoreTextLocY), basicAtlas)
	fmt.Fprintf(scoreTxt, "Score: %d", score)
	scoreTxt.Draw(win, pixel.IM.Scaled(scoreTxt.Orig, 2))

	nextPieceTextLocX := 142.0
	nextPieceTextLocY := 285.0
	nextPieceTxt := text.New(pixel.V(nextPieceTextLocX, nextPieceTextLocY), basicAtlas)
	fmt.Fprintf(nextPieceTxt, "Next Piece:")
	nextPieceTxt.Draw(win, pixel.IM)

	holdPieceTextLocX := 142.0
	holdPieceTextLocY := 385.0
	holdPieceTxt := text.New(pixel.V(holdPieceTextLocX, holdPieceTextLocY), basicAtlas)
	fmt.Fprintf(holdPieceTxt, "Hold Piece:")
	holdPieceTxt.Draw(win, pixel.IM)
}

func displayHoldPiece(win *pixelgl.Window) {
	if holdPiece == NoPiece {
		return
	}

	// Display hold piece
	baseShape := getShapeFromPiece(holdPiece)
	pic := blockGen(block2spriteIdx(piece2Block(holdPiece)))
	sprite := pixel.NewSprite(pic, pic.Bounds())
	boardBlockSize := 20.0
	scaleFactor := float64(boardBlockSize) / pic.Bounds().Max.Y
	shapeWidth := getShapeWidth(baseShape) + 1
	shapeHeight := 2

	holdPieceBGSprite.Draw(win, pixel.IM.Moved(pixel.V(182, 325)))

	for i := 0; i < 4; i++ {
		r := baseShape[i].row
		c := baseShape[i].col
		x := float64(c)*boardBlockSize + boardBlockSize/2
		y := float64(r)*boardBlockSize + boardBlockSize/2
		sprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scaleFactor).Moved(pixel.V(x+182-(float64(shapeWidth)*10), y+325-(float64(shapeHeight)*10))))
	}
}

func displayBG(win *pixelgl.Window) {
	// Display various background images
	bgImgSprite.Draw(win, pixel.IM.Moved(win.Bounds().Center()))
	gameBGSprite.Draw(win, pixel.IM.Moved(win.Bounds().Center()))
	nextPieceBGSprite.Draw(win, pixel.IM.Moved(pixel.V(182, 225)))

	// Display next block
	baseShape := getShapeFromPiece(nextPiece)
	pic := blockGen(block2spriteIdx(piece2Block(nextPiece)))
	sprite := pixel.NewSprite(pic, pic.Bounds())
	boardBlockSize := 20.0
	scaleFactor := float64(boardBlockSize) / pic.Bounds().Max.Y
	shapeWidth := getShapeWidth(baseShape) + 1
	shapeHeight := 2

	for i := 0; i < 4; i++ {
		r := baseShape[i].row
		c := baseShape[i].col
		x := float64(c)*boardBlockSize + boardBlockSize/2
		y := float64(r)*boardBlockSize + boardBlockSize/2
		sprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scaleFactor).Moved(pixel.V(x+182-(float64(shapeWidth)*10), y+225-(float64(shapeHeight)*10))))
	}
}

// block2spriteIdx associates a blocks color (b Block) with its index in the sprite sheet.
func block2spriteIdx(b Block) int {
	return int(b) - 1
}

// piece2Block associates a pieces shape (Piece) with it's color/image (Block).
func piece2Block(p Piece) Block {
	switch p {
	case LPiece:
		return Goluboy
	case IPiece:
		return Siniy
	case OPiece:
		return Pink
	case TPiece:
		return Purple
	case SPiece:
		return Red
	case ZPiece:
		return Yellow
	case JPiece:
		return Green
	}
	panic("piece2Block: Invalid piece passed in")
	return GraySpecial // Return strange value value
}

// initializeBag creates a new shuffled bag of all 7 pieces
func initializeBag() {
	pieceBag = make([]Piece, 7)

	// Fill the bag with one of each piece
	for i := 0; i < 7; i++ {
		pieceBag[i] = Piece(i)
	}

	// Shuffle the bag using Fisher-Yates algorithm
	for i := 6; i > 0; i-- {
		j := rand.Intn(i + 1)
		pieceBag[i], pieceBag[j] = pieceBag[j], pieceBag[i]
	}
}

// getNextPiece returns the next piece from the 7-bag
func getNextPiece() Piece {
	// If bag is empty or nil, create a new one
	if pieceBag == nil || len(pieceBag) == 0 {
		initializeBag()
	}

	// Take the first piece from the bag
	nextPiece := pieceBag[0]

	// Remove the first piece from the bag
	pieceBag = pieceBag[1:]

	return nextPiece
}

// Check if a T-spin was performed for scoring
func isTSpin(board Board) bool {
	// Only check for T-spins with T pieces
	if currentPiece != TPiece || !lastMovementWasRotation {
		return false
	}

	// For a T-spin, at least 3 of the 4 corners around the T's center must be blocked
	centerRow := activeShape[1].row
	centerCol := activeShape[1].col

	// Check each of the 4 corners around the T's center
	corners := [][2]int{
		{centerRow + 1, centerCol + 1}, // top-right
		{centerRow + 1, centerCol - 1}, // top-left
		{centerRow - 1, centerCol + 1}, // bottom-right
		{centerRow - 1, centerCol - 1}, // bottom-left
	}

	blockedCorners := 0
	for _, corner := range corners {
		r, c := corner[0], corner[1]
		// Check if corner is blocked (either by wall or another block)
		if r < 0 || r >= BoardRows || c < 0 || c >= BoardCols || board[r][c] != Empty {
			blockedCorners++
		}
	}

	// Require at least 3 corners to be blocked for a T-spin
	return blockedCorners >= 3
}
