package main

import (
	"math/rand"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

// isTouchingFloor checks if the piece that the user is controlling has a piece
// directly below it. Used to give the user more time when placing block on
// floor
func (b *Board) isTouchingFloor() bool {
	blockType := b[activeShape[0].row][activeShape[0].col]
	b.drawPiece(activeShape, Empty)
	isTouching := b.checkCollision(moveShapeDown(activeShape))
	b.drawPiece(activeShape, blockType)
	return isTouching
}

// rotatePiece rotates the piece that the user is currently moving.
// direction 1 for clockwise, -1 for counter-clockwise.
// Implements an ultra-responsive rotation system with generous wall kicks.
// Returns true if rotation succeeded, false otherwise.
func (b *Board) rotatePiece(direction int) bool {
	// The O piece should not be rotated
	if currentPiece == OPiece {
		return false
	}
	blockType := b[activeShape[0].row][activeShape[0].col]
	// Erase Piece
	b.drawPiece(activeShape, Empty)

	// Save the shape before rotation for T-spin detection
	lastRotationPoint = activeShape

	// Get the new shape based on rotation direction
	var newShape Shape
	if direction == 1 {
		newShape = rotateShape(activeShape)
	} else {
		newShape = rotateShapeCounterClockwise(activeShape)
	}

	// Try to place with standard wall kicks first
	kicks := wallKickData(currentPiece, rotationState, direction)
	rotated := false

	// Try standard kicks for all pieces
	for _, kick := range kicks {
		kickedShape := moveShape(kick[1], kick[0], newShape) // x, y offset
		if !b.checkCollision(kickedShape) {
			// Wall kick succeeded
			activeShape = kickedShape
			rotated = true

			// Update rotation state
			rotationState = (rotationState + direction) % 4
			if rotationState < 0 {
				rotationState += 4
			}

			// Set flag for T-spin detection
			lastMovementWasRotation = true
			break
		}
	}

	// If standard kicks failed, try extra kicks for ALL pieces, not just I
	if !rotated {
		// Get extra aggressive kicks
		extraKicks := getExtraIKicks(rotationState, direction)
		for _, kick := range extraKicks {
			kickedShape := moveShape(kick[1], kick[0], newShape)
			if !b.checkCollision(kickedShape) {
				// Extra kick succeeded
				activeShape = kickedShape
				rotated = true

				// Update rotation state
				rotationState = (rotationState + direction) % 4
				if rotationState < 0 {
					rotationState += 4
				}

				lastMovementWasRotation = true
				break
			}
		}
	}

	// If still not rotated, try one last set of aggressive kicks as a last resort
	if !rotated {
		// Extremely aggressive last resort kicks - will almost always find a spot
		lastResortKicks := [][2]int{
			{0, 4}, {4, 0}, {0, -4}, {-4, 0},  // Far kicks
			{3, 3}, {-3, 3}, {3, -3}, {-3, -3}, // Corner kicks
			{5, 0}, {0, 5}, {-5, 0}, {0, -5},  // Very far kicks
		}

		for _, kick := range lastResortKicks {
			kickedShape := moveShape(kick[1], kick[0], newShape)
			if !b.checkCollision(kickedShape) {
				// Last resort kick succeeded
				activeShape = kickedShape
				rotated = true

				// Update rotation state
				rotationState = (rotationState + direction) % 4
				if rotationState < 0 {
					rotationState += 4
				}

				lastMovementWasRotation = true
				break
			}
		}
	}

	if !rotated {
		// Failed to rotate with any wall kick
		b.drawPiece(activeShape, blockType)
		return false
	}

	b.drawPiece(activeShape, blockType)
	return true
}

// holdPiece allows the player to hold the current piece and retrieve a previously held piece
func (b *Board) holdPiece() {
	if !canHold {
		return
	}

	// Erase current piece
	b.drawPiece(activeShape, Empty)

	if holdPiece == NoPiece {
		// First hold - store current piece and get next piece
		holdPiece = currentPiece
		b.addPiece()
	} else {
		// Swap current piece with held piece
		tempPiece := holdPiece
		holdPiece = currentPiece

		// Create the held piece
		var offset int
		if tempPiece == IPiece {
			offset = rand.Intn(7)
		} else if tempPiece == OPiece {
			offset = rand.Intn(9)
		} else {
			offset = rand.Intn(8)
		}
		baseShape := getShapeFromPiece(tempPiece)
		baseShape = moveShape(20, offset, baseShape)
		b.fillShape(baseShape, piece2Block(tempPiece))
		currentPiece = tempPiece
		activeShape = baseShape
		rotationState = 0 // Reset rotation state for new piece
	}

	canHold = false // Prevent multiple holds until next piece
}

// lockPiece finalizes the current piece position and adds a new piece
func (b *Board) lockPiece() {
	if isGameOver(activeShape) {
		gameOver = true
		return
	}
	b.checkRowCompletion(activeShape)
	b.addPiece() // Replace with random piece
	canHold = true // Enable hold for the next piece
}

// movePiece attemps to move the piece that the user is controlling either
// right or left. +1 signifies a right move while -1 signifies a left move
func (b *Board) movePiece(dir int) bool {
	blockType := b[activeShape[0].row][activeShape[0].col]

	// Erase old piece for accurate collision detection
	b.drawPiece(activeShape, Empty)

	// Get the proposed new shape
	newShape := moveShape(0, dir, activeShape)

	// Check collision with optimized algorithm
	didCollide := b.checkCollision(newShape)

	if !didCollide {
		// Update to new position
		activeShape = newShape
		lastMovementWasRotation = false // Reset T-spin detection

		// Draw the piece at new position
		b.drawPiece(activeShape, blockType)
		return true // Successfully moved
	} else {
		// Movement failed due to collision - restore original position
		b.drawPiece(activeShape, blockType)
		return false
	}
}

// drawPiece sets the values of a board, b, to a specific block type, t
// according to shape, s.
func (b *Board) drawPiece(s Shape, t Block) {
	for i := 0; i < 4; i++ {
		b[activeShape[i].row][activeShape[i].col] = t
	}
}

// checkCollision checks if at the 4 points of a shape, s, there is
// nothing but Empty value under it and the position of the shape
// is inside the playing board (10x22 (top two rows invisiable)).
func (b Board) checkCollision(s Shape) bool {
	for i := 0; i < 4; i++ {
		r := s[i].row
		c := s[i].col
		if r < 0 || r > 21 || c < 0 || c > 9 || b[r][c] != Empty {
			return true
		}
	}
	return false
}

// applyGravity is the function that moves a piece down. If a collision
// is detected place the piece down and add a new piece. Returns wheather
// a collision was made.
func (b *Board) applyGravity() bool {
	blockType := b[activeShape[0].row][activeShape[0].col]
	// Erase old piece
	b.drawPiece(activeShape, Empty)

	// Does the block collide if it moves down?
	didCollide := b.checkCollision(moveShapeDown(activeShape))

	if !didCollide {
		activeShape = moveShapeDown(activeShape)
		lastMovementWasRotation = false // Reset T-spin detection
	}

	b.drawPiece(activeShape, blockType)

	return didCollide
}

// instafall calls the applyGravity function until a collision is detected.
func (b *Board) instafall() {
	collide := false
	for !collide {
		collide = b.applyGravity()
	}
	// Lock the piece immediately
	b.lockPiece()
}

// checkRowCompletion checks if the rows in a given shape are filled (ie should
// be deleted). If full, deletes the rows.
func (b *Board) checkRowCompletion(s Shape) {
	// Check for T-spin before any rows are deleted
	tSpin := isTSpin(*b)

	// Ony the rows of the shape can be filled
	rowWasDeleted := true
	// Since when we delete a row it can be shifted down, repeatedly try
	// to delete a row until no more deletes can be made
	var deleteRowCt int
	for rowWasDeleted {
		rowWasDeleted = false
		for i := 0; i < 4; i++ {
			r := s[i].row
			emptyFound := false
			// Look for empty row
			for c := 0; c < 10; c++ {
				if b[r][c] == Empty {
					emptyFound = true
					continue
				}
			}
			// If no empty cell was found in row delete row
			if !emptyFound {
				b.deleteRow(r)
				rowWasDeleted = true
				deleteRowCt++
			}
		}
	}

	// Score based on number of lines cleared and T-spin
	if deleteRowCt > 0 {
		// Base score for line clears
		baseScore := 100 * deleteRowCt

		// Bonus for multiple lines
		if deleteRowCt > 1 {
			baseScore *= deleteRowCt
		}

		// T-spin bonus (modern scoring)
		if tSpin {
			// T-spin bonus is 2x for a single, 3x for double, 4x for triple
			baseScore *= (deleteRowCt + 1)
			// Additional bonus for T-spin
			baseScore += 400
		}

		// Add to score
		score += baseScore
	} else if tSpin {
		// Mini T-spin (no lines cleared)
		score += 100
	}

	// Reset T-spin detection
	lastMovementWasRotation = false
}

// deleteRow remoes a row by shifting everything above it down by one.
func (b *Board) deleteRow(row int) {
	for r := row; r < 21; r++ {
		for c := 0; c < 10; c++ {
			b[r][c] = b[r+1][c]
		}
	}
}

// setPiece sets a value in the game board to a specific block type.
func (b *Board) setPiece(r, c int, val Block) {
	b[r][c] = val
}

// fillShape sets
func (b *Board) fillShape(s Shape, val Block) {
	for i := 0; i < 4; i++ {
		b.setPiece(s[i].row, s[i].col, val)
	}
}

// addPiece creates a piece at the top of the screen at a random position
// and sets it to the piece that the player is controlling
// (ie activeShape).
func (b *Board) addPiece() {
	var offset int
	if nextPiece == IPiece {
		offset = rand.Intn(7)
	} else if nextPiece == OPiece {
		offset = rand.Intn(9)
	} else {
		offset = rand.Intn(8)
	}
	baseShape := getShapeFromPiece(nextPiece)
	baseShape = moveShape(20, offset, baseShape)
	b.fillShape(baseShape, piece2Block(nextPiece))
	currentPiece = nextPiece
	activeShape = baseShape
	nextPiece = getNextPiece() // Use 7-bag system instead of random
	rotationState = 0 // Reset rotation state for new piece
}

// displayBoard displays a particular game board with all of its pieces
// onto a given window, win
func (b *Board) displayBoard(win *pixelgl.Window) {
	boardBlockSize := 20.0
	pic := blockGen(0)
	imgSize := pic.Bounds().Max.X
	scaleFactor := float64(boardBlockSize) / float64(imgSize)

	// Use consistent offsets for proper grid alignment
	const boardOffsetX = 282.0
	const boardOffsetY = 25.0

	// Create a map to cache sprites for each block type
	spriteCache := make(map[Block]*pixel.Sprite, 16)

	// First get the active shape and ghost shape
	pieceType := b[activeShape[0].row][activeShape[0].col]
	ghostShape := activeShape
	b.drawPiece(activeShape, Empty)
	for {
		if b.checkCollision(moveShapeDown(ghostShape)) {
			break
		}
		ghostShape = moveShapeDown(ghostShape)
	}
	b.drawPiece(activeShape, pieceType)

	// Draw board pieces directly
	for r := 0; r < 20; r++ {
		for c := 0; c < 10; c++ {
			if b[r][c] != Empty {
				// Get or create cached sprite
				spriteIdx := block2spriteIdx(b[r][c])
				sprite, exists := spriteCache[b[r][c]]
				if !exists {
					blockPic := blockGen(spriteIdx)
					sprite = pixel.NewSprite(blockPic, blockPic.Bounds())
					spriteCache[b[r][c]] = sprite
				}

				// Calculate position using consistent offsets
				x := float64(c)*boardBlockSize + boardBlockSize/2
				y := float64(r)*boardBlockSize + boardBlockSize/2

				// Apply visual feedback for active piece
				scale := scaleFactor
				if visualFeedbackActive && isPartOfActiveShape(r, c) {
					// Subtle scale pulse effect for tactile feedback
					pulseIntensity := 0.1 * (1.0 - (lastTapTime / 0.08))
					scale = scaleFactor * (1.0 + pulseIntensity)
				}

				sprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scale).Moved(pixel.V(x+boardOffsetX, y+boardOffsetY)))
			}
		}
	}

	// Draw ghost piece with transparency
	ghostBlockPic := blockGen(block2spriteIdx(pieceType))
	ghostSprite := pixel.NewSprite(ghostBlockPic, ghostBlockPic.Bounds())

	for i := 0; i < 4; i++ {
		r := ghostShape[i].row
		c := ghostShape[i].col

		// Only draw ghost if it doesn't overlap with active piece
		if !isPartOfActiveShape(r, c) && r < 20 {
			x := float64(c)*boardBlockSize + boardBlockSize/2
			y := float64(r)*boardBlockSize + boardBlockSize/2

			ghostSprite.DrawColorMask(win,
				pixel.IM.Scaled(pixel.ZV, scaleFactor).Moved(pixel.V(x+boardOffsetX, y+boardOffsetY)),
				pixel.RGBA{R: 1, G: 1, B: 1, A: 0.4})
		}
	}

	// Draw the active piece with emphasis
	for i := 0; i < 4; i++ {
		r := activeShape[i].row
		c := activeShape[i].col

		if r < 20 { // Only draw visible parts
			x := float64(c)*boardBlockSize + boardBlockSize/2
			y := float64(r)*boardBlockSize + boardBlockSize/2

			activePic := blockGen(block2spriteIdx(pieceType))
			activeSprite := pixel.NewSprite(activePic, activePic.Bounds())

			// Apply visual emphasis for active piece
			scale := scaleFactor
			if visualFeedbackActive {
				// Enhanced effect for active piece
				pulseIntensity := 0.15 * (1.0 - (lastTapTime / 0.08))
				scale = scaleFactor * (1.0 + pulseIntensity)
			}

			activeSprite.Draw(win, pixel.IM.Scaled(pixel.ZV, scale).Moved(pixel.V(x+boardOffsetX, y+boardOffsetY)))
		}
	}
}

// isPartOfActiveShape checks if a given position is part of the active shape
func isPartOfActiveShape(row, col int) bool {
	for i := 0; i < 4; i++ {
		if activeShape[i].row == row && activeShape[i].col == col {
			return true
		}
	}
	return false
}
