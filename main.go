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
	DASDelay           = 0.033 // Reduced initial delay for more responsive control
	ARRRate            = 0.033 // Faster repeat rate for better responsiveness
	ControlSensitivity = 0.05  // Longer window to detect quick taps
	SoftDropSpeed      = 0.05  // Faster soft drop speed for better responsiveness
	SoftDropFriction   = 0.1   // Less friction for smoother soft drops
	TapMovePriority    = true  // Always prioritize tap movement over DAS/ARR
	InputBufferWindow  = 0.1   // Input buffer window to capture inputs slightly early
)

var gameBoard Board
var activeShape Shape // The shape that the player controls
var currentPiece Piece
var gravityTimer float64
var baseSpeed float64 = 0.8
var gravitySpeed float64 = 0.8
var lockDelay float64 = 0.25 // Slightly increased for better placement opportunity
var lockDelayTimer float64 = 0
var lockResets int = 0
var maxLockResets int = 30
var levelUpTimer float64 = levelLength
var gameOver bool = false
var leftRightTimer float64
var ARRTimer float64
var lastMoveDirection int = 0
var keyReleaseTimer float64 = 0
var lastKeyReleaseTime float64 = 0
var isTapMovement bool = false
var inputBuffer map[pixelgl.Button]float64 = make(map[pixelgl.Button]float64) // New input buffer system
var score int
var nextPiece Piece
var holdPiece Piece = NoPiece
var canHold bool = true
var rotationState int = 0
var pieceBag []Piece = nil
var lastMovementWasRotation bool = false
var lastRotationPoint Shape
var rotationCooldown float64 = 0.0
var rotationDirection int = 0
var lastTapTime float64 = 0
var visualFeedbackActive bool = false
var softDropFrictionTimer float64 = 0
var lastSoftDropTime float64 = 0
var movementSmoothing bool = true // Enable movement smoothing for transitions

var blockGen func(int) pixel.Picture
var bgImgSprite pixel.Sprite
var gameBGSprite pixel.Sprite
var nextPieceBGSprite pixel.Sprite
var holdPieceBGSprite pixel.Sprite

func main() {
	// Ensure random number generator is seeded properly
	rand.Seed(time.Now().UnixNano())
	pixelgl.Run(run)
}

// run is the main code for the game. Allows pixelgl to run on main thread
func run() {
	// Initialize the window with minimum size constraints
	windowWidth := 765.0
	windowHeight := 450.0
	minWindowWidth := 640.0  // Minimum width to keep UI elements usable
	minWindowHeight := 400.0 // Minimum height to keep UI elements usable

	cfg := pixelgl.WindowConfig{
		Title:  "Blockfall",
		Bounds: pixel.R(0, 0, windowWidth, windowHeight),
		VSync:  true,
		// VSync will help limit refresh rate
		Monitor:   nil,
		Resizable: true, // Allow window resizing
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// Track initial/reference dimensions for scaling calculations
	initialWidth := windowWidth
	initialHeight := windowHeight

	// Store initial layout positions and sizes for responsive scaling
	const initialBoardOffsetX = 282.0
	const initialBoardOffsetY = 25.0
	const initialNextPieceX = 182.0
	const initialNextPieceY = 225.0
	const initialHoldPieceX = 182.0
	const initialHoldPieceY = 325.0
	const initialScoreX = 500.0
	const initialScoreY = 400.0
	const initialNextPieceTxtX = 142.0
	const initialNextPieceTxtY = 285.0
	const initialHoldPieceTxtX = 142.0
	const initialHoldPieceTxtY = 385.0

	// Track UI scale factor (will be updated based on window size)
	uiScaleFactor := 1.0

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

	// Set up frame limiter for consistent timing and reduced CPU usage
	const targetFPS = 120 // Increased FPS for smoother rendering
	frameDuration := time.Second / targetFPS
	last := time.Now()

	// Create and reuse text objects
	basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	scoreTxt := text.New(pixel.V(initialScoreX, initialScoreY), basicAtlas)
	nextPieceTxt := text.New(pixel.V(initialNextPieceTxtX, initialNextPieceTxtY), basicAtlas)
	holdPieceTxt := text.New(pixel.V(initialHoldPieceTxtX, initialHoldPieceTxtY), basicAtlas)

	// Store previous window size to detect changes
	prevWinWidth := win.Bounds().W()
	prevWinHeight := win.Bounds().H()

	for !win.Closed() && !gameOver {
		frameStart := time.Now()

		// Perform time processing events
		dt := time.Since(last).Seconds()
		last = time.Now()

		// Don't use too small time steps
		if dt > 0.25 {
			dt = 0.25 // Cap to reasonable value
		}

		// Check if window size changed and update scaling factors
		currWinWidth := win.Bounds().W()
		currWinHeight := win.Bounds().H()

		if currWinWidth != prevWinWidth || currWinHeight != prevWinHeight {
			// Enforce minimum window size by resizing the window if it's too small
			if currWinWidth < minWindowWidth || currWinHeight < minWindowHeight {
				newWidth := math.Max(currWinWidth, minWindowWidth)
				newHeight := math.Max(currWinHeight, minWindowHeight)
				win.SetBounds(pixel.R(
					win.Bounds().Min.X,
					win.Bounds().Min.Y,
					win.Bounds().Min.X+newWidth,
					win.Bounds().Min.Y+newHeight,
				))
				currWinWidth = newWidth
				currWinHeight = newHeight
			}

			// Recalculate UI scale factor based on the smaller dimension ratio to preserve aspect ratio
			widthRatio := currWinWidth / initialWidth
			heightRatio := currWinHeight / initialHeight

			// Use the smaller ratio to ensure everything fits
			uiScaleFactor = math.Min(widthRatio, heightRatio)

			// Update position of text elements for new window size
			scoreTxt = text.New(pixel.V(initialScoreX*widthRatio, initialScoreY*heightRatio), basicAtlas)
			nextPieceTxt = text.New(pixel.V(initialNextPieceTxtX*widthRatio, initialNextPieceTxtY*heightRatio), basicAtlas)
			holdPieceTxt = text.New(pixel.V(initialHoldPieceTxtX*widthRatio, initialHoldPieceTxtY*heightRatio), basicAtlas)

			// Update tracked window size
			prevWinWidth = currWinWidth
			prevWinHeight = currWinHeight
		}

		// Update input buffer - clear expired inputs
		for key, timestamp := range inputBuffer {
			timestamp -= dt
			if timestamp <= 0 {
				delete(inputBuffer, key)
			} else {
				inputBuffer[key] = timestamp
			}
		}

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
			gravityTimer = 0 // Reset completely for more consistent timing
			didCollide := gameBoard.applyGravity()
			if didCollide {
				score += 10
			}
		}

		// Speed up
		if levelUpTimer <= 0 {
			if baseSpeed > 0.1 {
				baseSpeed = math.Max(baseSpeed-speedUpRate, 0.1)
			}
			levelUpTimer = levelLength
			gravitySpeed = baseSpeed
		}

		// Input handling with prioritization and immediate response
		leftPressed := win.Pressed(pixelgl.KeyLeft)
		rightPressed := win.Pressed(pixelgl.KeyRight)

		// Buffer all new key presses for responsive control
		if win.JustPressed(pixelgl.KeyLeft) {
			inputBuffer[pixelgl.KeyLeft] = InputBufferWindow
			keyReleaseTimer = 0
			isTapMovement = true

			// Use the debounced movement system for consistent feel
			processMoveWithBounce(win, -1)
		}

		if win.JustPressed(pixelgl.KeyRight) {
			inputBuffer[pixelgl.KeyRight] = InputBufferWindow
			keyReleaseTimer = 0
			isTapMovement = true

			// Use the debounced movement system for consistent feel
			processMoveWithBounce(win, 1)
		}

		// Process key releases with improved tap detection
		if win.JustReleased(pixelgl.KeyLeft) || win.JustReleased(pixelgl.KeyRight) {
			lastKeyReleaseTime = 0

			// Short taps get special treatment for precision movement
			if keyReleaseTimer < ControlSensitivity {
				isTapMovement = false

				// Reset auto-repeat system to prevent unwanted movement
				leftRightTimer = DASDelay * 1.5 // Add a small delay after taps for better control
				ARRTimer = 0
			}
		}

		// Update tap detection timer
		if isTapMovement {
			keyReleaseTimer += dt
			if keyReleaseTimer > ControlSensitivity {
				isTapMovement = false // No longer considered a tap after sensitivity threshold
			}
		}

		// Determine movement direction with intelligent conflict resolution
		direction := 0
		if leftPressed && rightPressed {
			// If both keys are pressed, use the most recently pressed one based on buffer
			leftTime, hasLeft := inputBuffer[pixelgl.KeyLeft]
			rightTime, hasRight := inputBuffer[pixelgl.KeyRight]

			if hasLeft && hasRight {
				if leftTime > rightTime {
					direction = -1
				} else {
					direction = 1
				}
			} else if hasLeft {
				direction = -1
			} else if hasRight {
				direction = 1
			} else if lastMoveDirection != 0 {
				direction = lastMoveDirection
			}
		} else if leftPressed {
			direction = -1
		} else if rightPressed {
			direction = 1
		} else {
			// Reset DAS/ARR when no direction keys are pressed
			leftRightTimer = 0
			ARRTimer = 0
			lastMoveDirection = 0
		}

		// Handle movement with improved DAS/ARR system
		if direction != 0 {
			if direction != lastMoveDirection {
				// Direction change - immediate movement for responsiveness
				lastMoveDirection = direction
				leftRightTimer = DASDelay
				ARRTimer = 0

				// Only move here if we didn't already move in JustPressed
				if !win.JustPressed(pixelgl.KeyLeft) && !win.JustPressed(pixelgl.KeyRight) {
					processMoveWithBounce(win, direction)
				}
			} else if !isTapMovement {
				// Auto-shift handling for held keys
				leftRightTimer -= dt
				if leftRightTimer <= 0 {
					// DAS charged, use ARR for repeated movement
					ARRTimer += dt
					if ARRTimer >= ARRRate {
						// Reset ARR immediately for more consistent repeat rate
						ARRTimer = 0

						// Process movement with debouncing for smoother feel
						processMoveWithBounce(win, direction)
					}
				}
			}
		}

		// Update rotation cooldown
		if rotationCooldown > 0 {
			rotationCooldown -= dt
		}

		// Faster, more responsive soft drop
		if win.JustPressed(pixelgl.KeyDown) {
			gravitySpeed = SoftDropSpeed
			softDropFrictionTimer = 0
			lastSoftDropTime = 0

			// Immediate drop for responsiveness
			gameBoard.applyGravity()
		}

		if win.Pressed(pixelgl.KeyDown) {
			// More responsive soft drop system
			if softDropFrictionTimer > 0 {
				softDropFrictionTimer -= dt * 2 // Faster friction reduction
			}

			lastSoftDropTime += dt

			// More aggressive friction reduction for smoother continuous drops
			if lastSoftDropTime > 0.15 && softDropFrictionTimer > 0 {
				softDropFrictionTimer = 0 // Just clear it completely after a short delay
			}

			// Apply soft drop gravity with less friction
			if softDropFrictionTimer <= 0 {
				if gameBoard.applyGravity() {
					softDropFrictionTimer = SoftDropFriction
					lastSoftDropTime = 0
				}
			}
		}

		if win.JustReleased(pixelgl.KeyDown) {
			gravitySpeed = baseSpeed
			softDropFrictionTimer = 0
		}

		// More responsive rotation with reduced cooldown
		if win.JustPressed(pixelgl.KeyUp) {
			if rotationCooldown <= 0 {
				rotationSucceeded := gameBoard.rotatePiece(1) // Clockwise rotation
				if rotationSucceeded {
					rotationDirection = 1

					// Reset lock delay if rotated and on ground
					if gameBoard.isTouchingFloor() && lockResets < maxLockResets {
						lockDelayTimer = 0
						lockResets++
					}

					// Shorter rotation cooldown for more responsive feel
					rotationCooldown = 0.03
				}
			}
		}

		if win.JustPressed(pixelgl.KeyZ) {
			if rotationCooldown <= 0 {
				rotationSucceeded := gameBoard.rotatePiece(-1) // Counter-clockwise rotation
				if rotationSucceeded {
					rotationDirection = -1

					// Reset lock delay if rotated and on ground
					if gameBoard.isTouchingFloor() && lockResets < maxLockResets {
						lockDelayTimer = 0
						lockResets++
					}

					// Shorter rotation cooldown for more responsive feel
					rotationCooldown = 0.03
				}
			}
		}

		// More responsive hard drop
		if win.JustPressed(pixelgl.KeySpace) {
			// Skip the visual feedback drop and go straight to hard drop for immediate response
			preHardDropRow := activeShape[0].row
			gameBoard.instafall()

			// Scoring based on distance dropped
			dropDistance := preHardDropRow - activeShape[0].row
			score += 20 + dropDistance
		}

		// More responsive hold
		if win.JustPressed(pixelgl.KeyC) && canHold {
			gameBoard.holdPiece()
		}

		// Enhanced visual feedback
		if visualFeedbackActive {
			lastTapTime += dt
			if lastTapTime > 0.08 { // Shorter duration for snappier feedback
				visualFeedbackActive = false
			}
		}

		// Render at higher priority - move earlier in the frame
		win.Clear(colornames.Black)

		// Calculate center position based on current window dimensions
		windowCenter := win.Bounds().Center()

		// Draw backgrounds with responsive positioning
		// Background scales to fill entire window while maintaining aspect ratio
		bgScale := math.Max(win.Bounds().W()/bgImgSprite.Frame().W(), win.Bounds().H()/bgImgSprite.Frame().H())
		bgImgSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, bgScale).Moved(windowCenter))

		// Game board background scales based on UI scale factor
		gameScale := uiScaleFactor
		gameBGPos := pixel.V(windowCenter.X, windowCenter.Y)
		gameBGSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, gameScale).Moved(gameBGPos))

		// Next piece and hold piece background
		nextPiecePos := pixel.V(initialNextPieceX*uiScaleFactor, initialNextPieceY*uiScaleFactor)
		holdPiecePos := pixel.V(initialHoldPieceX*uiScaleFactor, initialHoldPieceY*uiScaleFactor)

		// Adjust positions based on window center offset
		xOffset := (win.Bounds().W() - initialWidth*uiScaleFactor) / 2
		yOffset := (win.Bounds().H() - initialHeight*uiScaleFactor) / 2

		nextPiecePos = nextPiecePos.Add(pixel.V(xOffset, yOffset))
		holdPiecePos = holdPiecePos.Add(pixel.V(xOffset, yOffset))

		nextPieceBGSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, uiScaleFactor).Moved(nextPiecePos))
		holdPieceBGSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, uiScaleFactor).Moved(holdPiecePos))

		// Display text content - reuse text objects with adjusted positions
		displayText(win, scoreTxt, nextPieceTxt, holdPieceTxt, uiScaleFactor)

		// Display game elements with responsive scaling
		displayHoldPiece(win, uiScaleFactor, xOffset, yOffset)
		displayNextPiece(win, uiScaleFactor, xOffset, yOffset)
		gameBoard.displayBoard(win)

		win.Update()

		// More responsive frame timing - minimize sleep when possible
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			sleepDuration := frameDuration - elapsed
			// Only sleep if we have more than 1ms to wait
			if sleepDuration > time.Millisecond {
				time.Sleep(sleepDuration)
			}
		}
	}
}

func displayText(win *pixelgl.Window, scoreTxt, nextPieceTxt, holdPieceTxt *text.Text, uiScaleFactor float64) {
	// Update and draw score
	scoreTxt.Clear()
	fmt.Fprintf(scoreTxt, "Score: %d", score)
	scoreTxt.Draw(win, pixel.IM.Scaled(scoreTxt.Orig, 2*uiScaleFactor))

	// Draw static text for next and hold pieces
	nextPieceTxt.Clear()
	fmt.Fprintf(nextPieceTxt, "Next Piece:")
	nextPieceTxt.Draw(win, pixel.IM.Scaled(nextPieceTxt.Orig, uiScaleFactor))

	holdPieceTxt.Clear()
	fmt.Fprintf(holdPieceTxt, "Hold Piece:")
	holdPieceTxt.Draw(win, pixel.IM.Scaled(holdPieceTxt.Orig, uiScaleFactor))
}

// Separate next piece display to its own function
func displayNextPiece(win *pixelgl.Window, uiScaleFactor float64, xOffset, yOffset float64) {
	baseShape := getShapeFromPiece(nextPiece)
	pic := blockGen(block2spriteIdx(piece2Block(nextPiece)))
	sprite := pixel.NewSprite(pic, pic.Bounds())
	boardBlockSize := 20.0 * uiScaleFactor
	scaleFactor := float64(boardBlockSize) / pic.Bounds().Max.Y
	shapeWidth := getShapeWidth(baseShape) + 1
	shapeHeight := 2

	initialNextPieceX := 182.0
	initialNextPieceY := 225.0

	for i := 0; i < 4; i++ {
		r := baseShape[i].row
		c := baseShape[i].col
		x := float64(c)*boardBlockSize + boardBlockSize/2
		y := float64(r)*boardBlockSize + boardBlockSize/2

		// Position calculation with scaling and offset
		posX := x + initialNextPieceX*uiScaleFactor - (float64(shapeWidth) * 10 * uiScaleFactor) + xOffset
		posY := y + initialNextPieceY*uiScaleFactor - (float64(shapeHeight) * 10 * uiScaleFactor) + yOffset

		sprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scaleFactor).Moved(pixel.V(posX, posY)))
	}
}

func displayHoldPiece(win *pixelgl.Window, uiScaleFactor float64, xOffset, yOffset float64) {
	if holdPiece == NoPiece {
		return
	}

	// Display hold piece
	baseShape := getShapeFromPiece(holdPiece)
	pic := blockGen(block2spriteIdx(piece2Block(holdPiece)))
	sprite := pixel.NewSprite(pic, pic.Bounds())
	boardBlockSize := 20.0 * uiScaleFactor
	scaleFactor := float64(boardBlockSize) / pic.Bounds().Max.Y
	shapeWidth := getShapeWidth(baseShape) + 1
	shapeHeight := 2

	initialHoldPieceX := 182.0
	initialHoldPieceY := 325.0

	// Draw the hold piece background with scaling
	holdPiecePos := pixel.V(initialHoldPieceX*uiScaleFactor+xOffset, initialHoldPieceY*uiScaleFactor+yOffset)
	holdPieceBGSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, uiScaleFactor).Moved(holdPiecePos))

	for i := 0; i < 4; i++ {
		r := baseShape[i].row
		c := baseShape[i].col
		x := float64(c)*boardBlockSize + boardBlockSize/2
		y := float64(r)*boardBlockSize + boardBlockSize/2

		// Position calculation with scaling and offset
		posX := x + initialHoldPieceX*uiScaleFactor - (float64(shapeWidth) * 10 * uiScaleFactor) + xOffset
		posY := y + initialHoldPieceY*uiScaleFactor - (float64(shapeHeight) * 10 * uiScaleFactor) + yOffset

		sprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scaleFactor).Moved(pixel.V(posX, posY)))
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
	// Always create a new slice to avoid issues with empty slices
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
		// Double check that bag was properly initialized
		if len(pieceBag) == 0 {
			// Emergency fallback - use a random piece if bag is still empty
			return Piece(rand.Intn(7))
		}
	}

	// Take the first piece from the bag
	nextPiece := pieceBag[0]

	// Remove the first piece from the bag
	if len(pieceBag) > 1 {
		pieceBag = pieceBag[1:]
	} else {
		// If this was the last piece, immediately refill the bag
		initializeBag()
	}

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

// isInputBuffered checks if a specific input is in the buffer and active
func isInputBuffered(key pixelgl.Button) bool {
	val, exists := inputBuffer[key]
	return exists && val > 0
}

// processMoveWithBounce processes directional movement with debouncing to prevent input stuttering
func processMoveWithBounce(win *pixelgl.Window, direction int) bool {
	// Always move at least once for snappy feel
	moveSucceeded := gameBoard.movePiece(direction)

	if moveSucceeded {
		lastTapTime = 0
		visualFeedbackActive = true

		// Reset lock delay if moved and on ground
		if gameBoard.isTouchingFloor() && lockResets < maxLockResets {
			lockDelayTimer = 0
			lockResets++
		}
		return true
	}

	return false
}
