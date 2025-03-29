package main

import "sync"

// Cache for rotated shapes to avoid recalculating them
var (
	rotationCacheMutex sync.RWMutex
	rotationCache     = make(map[Piece]map[int]map[int]Shape) // Piece -> rotationState -> direction -> Shape
)

// moveShape shifts a shape in a directy according to a given row and column.
func moveShape(r, c int, s Shape) Shape {
	var newShape Shape
	for i := 0; i < 4; i++ {
		newShape[i].row = s[i].row + r
		newShape[i].col = s[i].col + c
	}
	return newShape
}

func moveShapeDown(s Shape) Shape {
	return moveShape(-1, 0, s)
}

func moveShapeRight(s Shape) Shape {
	return moveShape(0, 1, s)
}

func moveShapeLeft(s Shape) Shape {
	return moveShape(0, -1, s)
}

// isGameOver checks if any of the Points in a shape are in the invisable rows
// (ie rows 20 and 21)
func isGameOver(s Shape) bool {
	for i := 0; i < 4; i++ {
		if s[i].row >= 20 {
			return true
		}
	}
	return false
}

func getShapeWidth(s Shape) int {
	maxWidth := 0
	for i := 1; i < 4; i++ {
		w := s[i].col - s[0].col
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func getShapeHeight(s Shape) int {
	maxHeight := -1
	minHeight := 22
	for i := 0; i < 4; i++ {
		if s[i].row < minHeight {
			minHeight = s[i].row
		}
		if s[i].row > maxHeight {
			maxHeight = s[i].row
		}
	}
	return maxHeight - minHeight
}

// rotateShape rotates a shape by 90 degrees based on the pivot point
// which is always the second element in the shape array (ie s[1]),
// except for the I piece which has a special pivot point.
func rotateShape(s Shape) Shape {
	// Special case: don't rotate O piece
	if currentPiece == OPiece {
		return s
	}

	// Check if the rotation is already cached
	rotationCacheMutex.RLock()
	if pieceCache, exists := rotationCache[currentPiece]; exists {
		if stateCache, exists := pieceCache[rotationState]; exists {
			if cachedShape, exists := stateCache[1]; exists {
				// Need to make a clean copy to avoid modifying cached shape
				var shapeCopy Shape
				copy(shapeCopy[:], cachedShape[:])
				// Adjust position based on the current shape's position

				// For I piece, the pivot is between blocks
				var offsetRow, offsetCol int
				if currentPiece == IPiece {
					// For I piece, use the center point between blocks 1 and 2 as pivot
					pivotRow := (s[1].row + s[2].row) / 2
					pivotCol := (s[1].col + s[2].col) / 2
					cachedPivotRow := (cachedShape[1].row + cachedShape[2].row) / 2
					cachedPivotCol := (cachedShape[1].col + cachedShape[2].col) / 2
					offsetRow = pivotRow - cachedPivotRow
					offsetCol = pivotCol - cachedPivotCol
				} else {
					offsetRow = s[1].row - cachedShape[1].row
					offsetCol = s[1].col - cachedShape[1].col
				}

				rotationCacheMutex.RUnlock()
				return moveShape(offsetRow, offsetCol, shapeCopy)
			}
		}
	}
	rotationCacheMutex.RUnlock()

	var retShape Shape

	if currentPiece == IPiece {
		// For I piece in SRS, the rotation center is between blocks
		// Calculate virtual center point between blocks 1 and 2
		pivotRow := (s[1].row + s[2].row) / 2
		pivotCol := (s[1].col + s[2].col) / 2

		// Perform rotation around this center point
		for i := 0; i < 4; i++ {
			dRow := s[i].row - pivotRow
			dCol := s[i].col - pivotCol
			retShape[i].row = pivotRow + (dCol * -1)
			retShape[i].col = pivotCol + dRow
		}
	} else {
		// For other pieces, use traditional rotation around block[1]
		pivot := s[1]
		retShape[1] = pivot
		for i := 0; i < 4; i++ {
			// Index 1 is the pivot point
			if i == 1 {
				continue
			}
			dRow := pivot.row - s[i].row
			dCol := pivot.col - s[i].col
			retShape[i].row = pivot.row + (dCol * -1)
			retShape[i].col = pivot.col + dRow
		}
	}

	// Cache this rotation for future use
	// Store only the basic shape (offset from 0,0) in the cache
	var offsetRow, offsetCol int
	if currentPiece == IPiece {
		// For I piece, normalize based on virtual center
		pivotRow := (retShape[1].row + retShape[2].row) / 2
		pivotCol := (retShape[1].col + retShape[2].col) / 2
		offsetRow = -pivotRow
		offsetCol = -pivotCol
	} else {
		offsetRow = -retShape[1].row
		offsetCol = -retShape[1].col
	}

	normalizedShape := moveShape(offsetRow, offsetCol, retShape)

	rotationCacheMutex.Lock()
	if _, exists := rotationCache[currentPiece]; !exists {
		rotationCache[currentPiece] = make(map[int]map[int]Shape)
	}
	if _, exists := rotationCache[currentPiece][rotationState]; !exists {
		rotationCache[currentPiece][rotationState] = make(map[int]Shape)
	}
	rotationCache[currentPiece][rotationState][1] = normalizedShape
	rotationCacheMutex.Unlock()

	return retShape
}

// rotateShapeCounterClockwise rotates a shape 90 degrees counter-clockwise
// based on the pivot point which is always the second element (s[1]),
// except for the I piece which has a special pivot point.
func rotateShapeCounterClockwise(s Shape) Shape {
	// Special case: don't rotate O piece
	if currentPiece == OPiece {
		return s
	}

	// Check if the rotation is already cached
	rotationCacheMutex.RLock()
	if pieceCache, exists := rotationCache[currentPiece]; exists {
		if stateCache, exists := pieceCache[rotationState]; exists {
			if cachedShape, exists := stateCache[-1]; exists {
				// Need to make a clean copy to avoid modifying cached shape
				var shapeCopy Shape
				copy(shapeCopy[:], cachedShape[:])

				// For I piece, the pivot is between blocks
				var offsetRow, offsetCol int
				if currentPiece == IPiece {
					// For I piece, use the center point between blocks 1 and 2 as pivot
					pivotRow := (s[1].row + s[2].row) / 2
					pivotCol := (s[1].col + s[2].col) / 2
					cachedPivotRow := (cachedShape[1].row + cachedShape[2].row) / 2
					cachedPivotCol := (cachedShape[1].col + cachedShape[2].col) / 2
					offsetRow = pivotRow - cachedPivotRow
					offsetCol = pivotCol - cachedPivotCol
				} else {
					offsetRow = s[1].row - cachedShape[1].row
					offsetCol = s[1].col - cachedShape[1].col
				}

				rotationCacheMutex.RUnlock()
				return moveShape(offsetRow, offsetCol, shapeCopy)
			}
		}
	}
	rotationCacheMutex.RUnlock()

	var retShape Shape

	if currentPiece == IPiece {
		// For I piece in SRS, the rotation center is between blocks
		// Calculate virtual center point between blocks 1 and 2
		pivotRow := (s[1].row + s[2].row) / 2
		pivotCol := (s[1].col + s[2].col) / 2

		// Perform rotation around this center point
		for i := 0; i < 4; i++ {
			dRow := s[i].row - pivotRow
			dCol := s[i].col - pivotCol
			retShape[i].row = pivotRow + dCol
			retShape[i].col = pivotCol + (dRow * -1)
		}
	} else {
		// For other pieces, use traditional rotation around block[1]
		pivot := s[1]
		retShape[1] = pivot
		for i := 0; i < 4; i++ {
			// Index 1 is the pivot point
			if i == 1 {
				continue
			}
			dRow := pivot.row - s[i].row
			dCol := pivot.col - s[i].col
			retShape[i].row = pivot.row + dCol
			retShape[i].col = pivot.col + (dRow * -1)
		}
	}

	// Cache this rotation for future use
	// Store only the basic shape (offset from 0,0) in the cache
	var offsetRow, offsetCol int
	if currentPiece == IPiece {
		// For I piece, normalize based on virtual center
		pivotRow := (retShape[1].row + retShape[2].row) / 2
		pivotCol := (retShape[1].col + retShape[2].col) / 2
		offsetRow = -pivotRow
		offsetCol = -pivotCol
	} else {
		offsetRow = -retShape[1].row
		offsetCol = -retShape[1].col
	}

	normalizedShape := moveShape(offsetRow, offsetCol, retShape)

	rotationCacheMutex.Lock()
	if _, exists := rotationCache[currentPiece]; !exists {
		rotationCache[currentPiece] = make(map[int]map[int]Shape)
	}
	if _, exists := rotationCache[currentPiece][rotationState]; !exists {
		rotationCache[currentPiece][rotationState] = make(map[int]Shape)
	}
	rotationCache[currentPiece][rotationState][-1] = normalizedShape
	rotationCacheMutex.Unlock()

	return retShape
}

// getShapeFromPiece returns the shape based on the piece type. There
// are seven shapes available: LPiece, IPiece, OPiece, TPiece, SPiece,
// ZPiece, and JPiece.
func getShapeFromPiece(p Piece) Shape {
	var retShape Shape
	switch p {
	case LPiece:
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 1, col: 1},
			Point{row: 1, col: 2},
			Point{row: 0, col: 0},
		}
	case IPiece:
		// In SRS, the I piece should have its pivot point centered
		// The blocks are arranged horizontally in the initial position
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 1, col: 1},
			Point{row: 1, col: 2},
			Point{row: 1, col: 3},
		}
	case OPiece:
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 1, col: 1},
			Point{row: 0, col: 0},
			Point{row: 0, col: 1},
		}
	case TPiece:
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 1, col: 1},
			Point{row: 1, col: 2},
			Point{row: 0, col: 1},
		}
	case SPiece:
		retShape = Shape{
			Point{row: 0, col: 0},
			Point{row: 0, col: 1},
			Point{row: 1, col: 1},
			Point{row: 1, col: 2},
		}
	case ZPiece:
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 1, col: 1},
			Point{row: 0, col: 1},
			Point{row: 0, col: 2},
		}
	case JPiece:
		retShape = Shape{
			Point{row: 1, col: 0},
			Point{row: 0, col: 1},
			Point{row: 0, col: 0},
			Point{row: 0, col: 2},
		}
	default:
		panic("getShapeFromPiece(Piece): Invalid piece entered")
	}
	return retShape
}

// wallKickData returns the wall kick offsets to test for the given piece and rotation.
// According to SRS (Super Rotation System) rules, but with enhanced kicks for better responsiveness.
// state is the current rotation state (0-3), where:
// 0 = spawn state, 1 = rotated right once, 2 = rotated twice, 3 = rotated left once
// direction is 1 for clockwise, -1 for counter-clockwise
func wallKickData(piece Piece, state int, direction int) [][2]int {
	// Get the new state based on direction
	newState := (state + direction) % 4
	if newState < 0 {
		newState += 4
	}

	// Different wall kick data for I piece vs JLSTZ pieces
	if piece == IPiece {
		// Extremely generous I piece wall kick data for responsive gameplay
		// Far more kick attempts than standard SRS
		kicksClockwise := [][][2]int{
			// 0->R (top row to right)
			{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}, {-2, 2}, {1, -2}, {3, 0}, {-3, 0}, {2, 3}, {-2, -3}},
			// R->2 (right to bottom)
			{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}, {-2, -2}, {3, 1}, {3, -1}, {-3, -1}, {0, 3}, {0, -3}},
			// 2->L (bottom to left)
			{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}, {2, -2}, {-3, 0}, {3, 2}, {-1, -3}, {4, 0}, {-4, 0}},
			// L->0 (left to top)
			{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}, {2, 2}, {-3, 1}, {-3, -3}, {3, -1}, {0, 3}, {0, -3}},
		}

		kicksCounterClockwise := [][][2]int{
			// 0->L (top row to left)
			{{0, 0}, {-1, 0}, {2, 0}, {-1, 2}, {2, -1}, {-2, 2}, {3, 0}, {1, -3}, {-3, 1}, {3, 3}, {-3, -3}},
			// R->0 (right to top)
			{{0, 0}, {2, 0}, {-1, 0}, {2, 1}, {-1, -2}, {-2, -2}, {3, 2}, {-3, 0}, {1, 3}, {3, -3}, {-3, 3}},
			// 2->R (bottom to right)
			{{0, 0}, {1, 0}, {-2, 0}, {1, -2}, {-2, 1}, {2, -2}, {-3, -1}, {3, 0}, {-1, 3}, {4, 0}, {-4, 0}},
			// L->2 (left to bottom)
			{{0, 0}, {-2, 0}, {1, 0}, {-2, -1}, {1, 2}, {2, 2}, {-3, 0}, {3, -2}, {-1, -3}, {0, 3}, {0, -3}},
		}

		if direction == 1 {
			return kicksClockwise[state]
		} else {
			return kicksCounterClockwise[state]
		}
	} else if piece != OPiece { // JLSTZ pieces (O piece doesn't rotate)
		// Enhanced JLSTZ pieces wall kick data
		kicksClockwise := [][][2]int{
			// 0->R
			{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}, {-2, 0}, {-2, 1}, {0, -3}, {-1, -3}, {-2, -2}, {2, 0}},
			// R->2
			{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}, {2, 0}, {2, -1}, {0, 3}, {1, 3}, {2, 2}, {-2, 0}},
			// 2->L
			{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}, {2, 0}, {2, 1}, {0, -3}, {1, -3}, {2, -2}, {-2, 0}},
			// L->0
			{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}, {-2, 0}, {-2, -1}, {0, 3}, {-1, 3}, {-2, 2}, {2, 0}},
		}

		kicksCounterClockwise := [][][2]int{
			// 0->L
			{{0, 0}, {1, 0}, {1, 1}, {0, -2}, {1, -2}, {2, 0}, {2, 1}, {0, -3}, {1, -3}, {2, -2}, {-2, 0}},
			// R->0
			{{0, 0}, {-1, 0}, {-1, -1}, {0, 2}, {-1, 2}, {-2, 0}, {-2, -1}, {0, 3}, {-1, 3}, {-2, 2}, {2, 0}},
			// 2->R
			{{0, 0}, {-1, 0}, {-1, 1}, {0, -2}, {-1, -2}, {-2, 0}, {-2, 1}, {0, -3}, {-1, -3}, {-2, -2}, {2, 0}},
			// L->2
			{{0, 0}, {1, 0}, {1, -1}, {0, 2}, {1, 2}, {2, 0}, {2, -1}, {0, 3}, {1, 3}, {2, 2}, {-2, 0}},
		}

		if direction == 1 {
			return kicksClockwise[state]
		} else {
			return kicksCounterClockwise[state]
		}
	}

	// O piece doesn't need wall kicks
	return [][2]int{{0, 0}}
}

// getExtraIKicks provides additional wall kick options for the I piece
// beyond the standard SRS kicks to make rotation feel more responsive
func getExtraIKicks(state int, direction int) [][2]int {
	// These are additional kick options that are not in standard SRS
	// but help make the I piece rotation feel more responsive
	clockwiseExtraKicks := [][][2]int{
		// 0->R - try kicks up to 4 spaces in all directions!
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// R->2
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// 2->L
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// L->0
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
	}

	counterClockwiseExtraKicks := [][][2]int{
		// 0->L
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// R->0
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// 2->R
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
		// L->2
		{{-3, 3}, {3, 3}, {3, -3}, {-3, -3}, {4, 2}, {4, -2}, {-4, 2}, {-4, -2}, {2, 4}, {2, -4}, {-2, 4}, {-2, -4}},
	}

	if direction == 1 {
		return clockwiseExtraKicks[state]
	} else {
		return counterClockwiseExtraKicks[state]
	}
}
