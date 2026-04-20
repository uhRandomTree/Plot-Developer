package main

import(
	"fmt"
	"math/rand/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	_ "image/png"
	"os"
	"time"
)

type Game struct{}

func GetTiles() (TileX, TileY int) {
	ClickedPosX, ClickedPosY := ebiten.CursorPosition()		
	TileX = ClickedPosX / TileSizeX
	TileY = ClickedPosY / TileSizeY
	return
}

func Dig(TileX, TileY int) {
	if displayBoard[TileX][TileY] != 10 { return }
	switch proximityBoard[TileX][TileY] {
			case 0:
				displayBoard[TileX][TileY] = 0
				exploreEmpty(TileX, TileY, TileX, TileY)

			case 9:
				ticker.Stop()
				fmt.Printf("\x1b[1A\x1b[2KTime: %.2f\x1b[1B\r", time.Now().Sub(firstClickTime).Seconds() )
				fmt.Println("GAME OVER")
				gameState++
				displayBoard[TileX][TileY] = 12
				for H := range Width {
					for V := range Height {
						if proximityBoard[H][V] == 9 {
							if displayBoard[H][V] == 10 { displayBoard[H][V] = 9 }
						} else if displayBoard[H][V] == 11 {
							displayBoard[H][V] = 13
						}
					}
				}
			default:
				displayBoard[TileX][TileY] = proximityBoard[TileX][TileY]
			}
}

func Flag(TileX, TileY int) {
	if displayBoard[TileX][TileY] == 10 {
		displayBoard[TileX][TileY] = 11
		flagged++
	} else if displayBoard[TileX][TileY] == 11 {
		displayBoard[TileX][TileY] = 10
		flagged--
	}
	// I may want to make this some sort of channel?
	fmt.Printf("\x1b[2A\x1b[2KBombs: %d\x1b[2B\r", Bombs-flagged)
}

func Sweep(TileX, TileY int) {
	// Should probably have a "selectedtile" var here or smth.
	if proximityBoard[TileX][TileY] > 7 || proximityBoard[TileX][TileY] == 0 { return } //If it's less than 0, it should only be revealed as everything around it is cleared, so not needed. Obviously, uncovering a bomb ends the game. And if the tile is an 8, there's no point to counting up the tiles, since it won't have anywhere to dig.
		
	var targetFlagNum = proximityBoard[TileX][TileY]
	for _, Dropper := range inBoundsTilesAround(TileX, TileY) {
		if displayBoard[Dropper.X][Dropper.Y] == 11 { targetFlagNum-- }
	}
	if targetFlagNum == 0 {
		for _, dropper := range inBoundsTilesAround(TileX, TileY) {
			if displayBoard[dropper.X][dropper.Y] == 10 {
				Dig(dropper.X, dropper.Y)
			}
		}
	}
}

func exploreEmpty (Xclicked, Yclicked, Xfrom, Yfrom int) {
	var H, V int
	for _, dropper := range inBoundsTilesAround(Xclicked, Yclicked) {
		H, V = dropper.X, dropper.Y
		if displayBoard[H][V] == 10 {
			displayBoard[H][V] = proximityBoard[H][V]
			// Could try to insert the tile counter here, it depends on if this recounts itself.
			if proximityBoard[H][V] == 0 { exploreEmpty(H, V, Xclicked, Yclicked) } 
		}
	}
}


var (
	flagged int
	gameState int8 // 0 is on first click, 1 is actively playing, 2 is over
	ResetTo int
	ticker *time.Ticker
	tickerDisplay int
	firstClickTime time.Time
)

type Action struct {
	IsHeld bool
	WasHeld bool // Last tick
	UsesButtons bool
	UsesKeys bool
	Buttons ebiten.MouseButton
	Keys ebiten.Key
}
// I'm sure something clever could be done with channels

var(
	Digging = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton0,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	Sweeping = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton1,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	Flagging = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton2,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	Resetting = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton4,
		UsesKeys: true,
		Keys: ebiten.KeyR,
	}
	// Should I bother making a quitting action?
	// Probably :(
	// ebiten.KeyNumpad0 will be the "dummy" key: it still works, it's just there to display that KB keys can't be used.
)

func (Minefield *Game) Update() error {
	for _, A := range [4]*Action{&Digging, &Sweeping, &Flagging, &Resetting} { // Are pointers correct here?
		A.WasHeld = A.IsHeld
		A.IsHeld = false
		if A.UsesButtons {
			if ebiten.IsMouseButtonPressed(A.Buttons) { A.IsHeld = true }
		}
		if A.UsesKeys {
			if ebiten.IsKeyPressed(A.Keys) { A.IsHeld = true }
		}
	}

	if Resetting.IsHeld {
		if ebiten.IsKeyPressed(ebiten.Key1) { ResetTo = 1 }
		if ebiten.IsKeyPressed(ebiten.Key2) { ResetTo = 2 }
		if ebiten.IsKeyPressed(ebiten.Key3) { ResetTo = 3 }
		if ebiten.IsKeyPressed(ebiten.Key4) { ResetTo = 4 }
		// These can be action keys, but I just don't think it matters. The user is resetting anyways.
	}

	if !Resetting.IsHeld && Resetting.WasHeld {
		// RESET
		gameState = 0
		flagged = 0
		SetDifficulty(ResetTo)
		if ResetTo == 4 { ResetTo = 0 }
		displayBoard = iniDisplayBoard(Width, Height)
		ticker.Stop()
		fmt.Println("RESETTING GAME")
		fmt.Printf("Bombs: %d\nTime: 0\n", Bombs)
	}
	
	if gameState == 2 { return nil }

	if Digging.IsHeld && Flagging.IsHeld { // Chording. Should I have a special case where I look at the actual keys? What maps better in the brain lol.
		Digging.IsHeld = false ; Flagging.IsHeld = false
		Sweeping.IsHeld = true
	}
	// check if the above works
	// Also check for the other combos.
	if !Digging.IsHeld && Digging.WasHeld {
		if gameState == 0 {
			ClX, ClY := GetTiles()
			// I imagine there's a better way to do this
			proximityBoard, bombBoard = iniGameBoards(ClX, ClY, Width, Height, Bombs)
			firstClickTime = time.Now()
			gameState++
			BechtelValue()
			// Start counting timing here
			ticker = time.NewTicker( time.Second )
			tickerDisplay = 1

			// Visual ticker
			go func() {
				for {
					<- ticker.C
					fmt.Printf("\x1b[1A\x1b[2KTime: %d\x1b[1B\r", tickerDisplay)
					tickerDisplay++
				}
			}()
		}
		// Get the hollow tiles
		Dig(GetTiles())
	}
	if !Sweeping.IsHeld && Sweeping.WasHeld { Sweep(GetTiles()) } // Hollow tiles
	if Flagging.IsHeld && !Flagging.WasHeld { Flag(GetTiles()) }

	
	var uncleared int
	for H := range Width {
		for V := range Height {
			if displayBoard[H][V] > 9 { uncleared++ }
		}
	}
	if uncleared == Bombs {
		fmt.Printf("\x1b[1A\x1b[2KTime: %.2f\x1b[1B\r", time.Since(firstClickTime).Seconds() )
		fmt.Println("3BV: ", BechtelValue())
		fmt.Println("YOU WIN")
		ticker.Stop()
		gameState++
	}
	// Ideally, I'd count up as tiles are cleared: it's probably more efficient than this.
	return nil
}

func (Minefield *Game) Layout(RealWidth, RealHeight int) (LogicalWidth, LogicalHeight int){
	return TileSizeX * Width, TileSizeY * Height
}

var(
Height, Width, Bombs, TileSizeX, TileSizeY int
proximityBoard, displayBoard [][]int8
Error error
Index *ebiten.Image
Location ebiten.DrawImageOptions
bombBoard [][]bool
)

type coord struct {
	X, Y int
}

func inBoundsTilesAround(Xclicked, Yclicked int) (Locations []coord) {
	// Is there a way to make this some sort of iterator?
	// Assumes the clicked tile is inbounds
	for H := range 3 {
		H += Xclicked - 1
		if (H < 0) || (H == Width) {continue}
		for V := range 3 {
			if (H == Xclicked) && (V == 1) {continue}
			V += Yclicked - 1
			if (V < 0) || (V == Height) {continue}
			Locations = append(Locations, coord{H, V})
		}
	}
	return
}

func iniDisplayBoard (Width, Height int) (DisplayBoard [][]int8) {
	DisplayBoard = make([][]int8, Width)
	for i := range Width {
		DisplayBoard[i] = make([]int8, Height)
		
		for Tiler := range Height {
			DisplayBoard[i][Tiler] = 10
		}
	}
	const ScaleConst int = 2
	ebiten.SetWindowSize(TileSizeX*Width*ScaleConst, TileSizeY*Height*ScaleConst) //to whatever real size we want it to display as.
	return DisplayBoard
}

func iniGameBoards (Xclicked, Yclicked, Width, Height, Bombs int) (ProximityBoard [][]int8, BombBoard [][]bool) {

	BombBoard = make([][]bool, Width)
	ProximityBoard = make([][]int8, Width)
	
	for i := range Width {
		BombBoard[i] = make([]bool, Height)
		ProximityBoard[i] = make([]int8, Height)
	}
	
	var BombPlace int = (Yclicked * Width) + Xclicked
	var BombLocation, BombLocX, BombLocY int
	var Spaces = Width * Height
	
	SRBombs := make([]int, Spaces)
	Spaces--
	for i := range Spaces { SRBombs[i] = i } // Can probably be done in one line, I don't know.

	SRBombs = append(SRBombs[:BombPlace], SRBombs[BombPlace+1:]...)
	
	for i := range Bombs {
		BombPlace = rand.IntN(Spaces - i)
		BombLocation = SRBombs[BombPlace]
		
		SRBombs = append( SRBombs[:BombPlace], SRBombs[BombPlace+1:]... )
		BombLocX, BombLocY = BombLocation % Width, BombLocation / Width
		BombBoard [ BombLocX ] [ BombLocY ] = true

		for _, validLocs := range inBoundsTilesAround(BombLocX, BombLocY) {
			if ProximityBoard[validLocs.X][validLocs.Y] < 9 {
				ProximityBoard[validLocs.X][validLocs.Y]++
			}
		}

		ProximityBoard [ BombLocX ] [ BombLocY ] = 9

	}
	return ProximityBoard, BombBoard
}

func BechtelValue() (Clicks int) {
	// Implementation slightly inspired by:
	// https://gamedev.stackexchange.com/questions/63046/how-should-i-calculate-the-score-in-minesweeper-3bv-or-3bv-s
	var Cleared = make([][]bool, Width)
	for i := range Width {
		Cleared[i] = make([]bool, Height)
	}
	
	for V := range Height {
		for H := range Width {
			if Cleared[H][V] { continue }
			Cleared[H][V] = true
			switch proximityBoard[H][V] {
				case 9: continue // Bombs aren't counted, obviously
				case 0: // This is where I have to do the flood fill sweeping thing.
					aroundZero := false // I deviated from the implementation here, but I thought this was clever.
					for _, Surrounding := range inBoundsTilesAround(H, V) {
						if Cleared[Surrounding.X][Surrounding.Y] && proximityBoard[Surrounding.X][Surrounding.Y] == 0 {
							Cleared[H][V] = true
							aroundZero = true
							continue
						}
					}
					if !aroundZero { Clicks++ }
					// These are very similar, I should do something about that.
				default:
					aroundZero := false
					for _, i := range inBoundsTilesAround(H, V) {
						if proximityBoard[i.X][i.Y] == 0 { aroundZero = true ; continue }
					}
					if !aroundZero { Clicks++ }	
					
			}
		}			
	}
	return Clicks // Could (should?) do a naked return, but better for readability.
}

func SetDifficulty(Chosen int) {
	// 1..4: Beginner 8x8 10, Intermediate 16x16 40, Expert 30x16 99, Custom WxH B
	switch Chosen {
		case 1 : Width, Height, Bombs = 8, 8, 10
		case 2 : Width, Height, Bombs = 16, 16, 40
		case 3 : Width, Height, Bombs = 30, 16, 99
		case 4 :
			fmt.Print("In the format WxH B: ")
			fmt.Scanf("%dx%d %d", &Width, &Height, &Bombs)// Does this need a \n?
			// I should probably check for errors
			for Width <= 0 {
				fmt.Printf("Dimensions must be >= 0.\nWidth: ")
				fmt.Scanf("%d\n", &Width)
			}
			for Height <= 0 {
				fmt.Printf("Dimensions must be >= 0.\nHeight: ")
				fmt.Scanf("%d\n", &Height)
			}
			for Bombs >= Width * Height {
				fmt.Printf("Bombs must be less than Width * Height.\nBombs: ")
				fmt.Scanf("%d\n", &Bombs)
			}
			// if Bombs > (Width - 1) * (Height - 1) {fmt.Println("Warning: ")}
	}
}

func main() {
	ebiten.SetScreenClearedEveryFrame(false)

	var Theme string = "ClassicXP"
	
	IndexFile, Error := os.Open(Theme + ".png") // Check if this is even openable/exists?
	defer IndexFile.Close() // I should check for if it fails to close.
	if Error != nil { fmt.Println(Error) }
	
	IndexImage, _, Error := image.Decode(IndexFile)
	if Error != nil { fmt.Println(Error) }
	Index = ebiten.NewImageFromImage(IndexImage)

	TileSizeX, TileSizeY = Index.Bounds().Max.X, Index.Bounds().Max.Y
	if TileSizeY % 14 != 0 {
		fmt.Println("Malformed tileset error")
		//FIGURE OUT WAY TO END GAME
	} else {
		TileSizeY /= 14
	}
	
	fmt.Printf(
`Select difficulty:
1. (B)eginner     8x8   10
2. (I)ntermediate 16x16 40
3. (E)xpert       30x16 99
4. (C)ustom       WxH   B
`)
	var Difficulty rune
	InvalidDifficultyEntered:
	_, Error = fmt.Scanf("%c\n", &Difficulty)
	// fmt.Scanf("%c\n", &Difficulty)
	// Sterilize this
	if Error != nil { fmt.Println(Error) }
	switch Difficulty {
		case 'B', 'b', '1' : SetDifficulty(1)
		case 'I', 'i', '2' : SetDifficulty(2)
		case 'E', 'e', '3' : SetDifficulty(3)
		case 'C', 'c', '4' : SetDifficulty(4)
		default:
			fmt.Println("Enter a valid difficulty, by # or (case insensitive) letter.")
			goto InvalidDifficultyEntered
	}
	fmt.Printf("Bombs: %d\nTime: 0\n", Bombs)

	// Difficulties:
	// Beginner - 8x8, 10
	// Intermediate - 16x16, 40
	// Expert - 30x16, 99
	// 3BV limits is 2, 30, 100

	displayBoard = iniDisplayBoard(Width, Height)
	// InitGameBoards runs on the first click; check in Update for when GameState == 0
	// Board is in (X, Y), starts at 0, 0 at top left.
	// Other work needs to know the location of the first click.
	
	ticker = time.NewTicker( time.Second )
	// This is so that if the user resets before they've clicked, there's a ticker to stop. Otherwise, I'm running .Stop() on something that doesn't exist.

	ebiten.SetWindowTitle("Minesweeper Clone")
	if Error = ebiten.RunGame( &Game{} ) ; Error != nil {
		fmt.Println(Error)
	}
}

func (Minefield *Game)  Draw(Screen *ebiten.Image) {	
	for Hori := range Width {
		for Vert := range Height {
			Location.GeoM.Reset()
			Location.GeoM.Translate(float64(Hori*TileSizeX), float64(Vert*TileSizeY))
			tileIndex := int(displayBoard[Hori][Vert])

			// Hollow tiles
			// Put here because I don't want to have to track or modify the DisplayBoard
			// xt, yt := GetTiles()
			if Digging.IsHeld {
				if xt, yt := GetTiles() ; Hori == xt && Vert == yt {
					if tileIndex == 10 {
						tileIndex = 0
					}
				}
			}

			if Sweeping.IsHeld {
				for _, validLocs := range inBoundsTilesAround(GetTiles()) {
					if Hori == validLocs.X && Vert == validLocs.Y {
						if tileIndex == 10 {
							tileIndex = 0
						}
					}
				}
			}
			
			// if xt, yt := GetTiles() ; Digging.isHeld && Hori == xt && Vert == yt && TileIndex == 10 { TileIndex = 0 }
			// Might be better? I don't know.
			Rect := image.Rectangle{
				image.Point{0, (TileSizeY * tileIndex)},
				image.Point{TileSizeX, (TileSizeY * (tileIndex + 1))}}
	
			//DisplayBoard[Hori][Vert] contains the "index" of tile we need to render.
			TileToDraw := Index.SubImage(Rect).(*ebiten.Image)
			Screen.DrawImage(TileToDraw, &Location)
		}
	}
}
