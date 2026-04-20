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

type Game struct{
	flagged int
	gameState int8 // 0 is on first click, 1 is actively playing, 2 is over
	ResetTo int
	ticker *time.Ticker
	tickerDisplay int
	firstClickTime time.Time
	Height, Width, Bombs, TileSizeX, TileSizeY int
	proximityBoard, displayBoard [][]int8
	Error error
	index *ebiten.Image
	Location ebiten.DrawImageOptions
	bombBoard [][]bool

	Digging, Sweeping, Flagging, Resetting Action
	// Should I bother making a quitting action?
	// Probably :(
	// Defined in main()
}

type Action struct {
	IsHeld bool
	WasHeld bool // Last tick
	UsesButtons bool
	UsesKeys bool
	Buttons ebiten.MouseButton
	Keys ebiten.Key
}
// I'm sure something clever could be done with channels

func (game *Game) GetTiles() (tileX, tileY int) {
	ClickedPosX, ClickedPosY := ebiten.CursorPosition()		
	return ClickedPosX / game.TileSizeX, ClickedPosY / game.TileSizeY
}

func (game *Game) Dig(TileX, TileY int) {
	if game.displayBoard[TileX][TileY] != 10 { return }
	switch game.proximityBoard[TileX][TileY] {
			case 0:
				game.displayBoard[TileX][TileY] = 0
				game.exploreEmpty(TileX, TileY, TileX, TileY)

			case 9:
				game.ticker.Stop()
				fmt.Printf("\x1b[1A\x1b[2KTime: %.2f\x1b[1B\r", time.Now().Sub(game.firstClickTime).Seconds() )
				fmt.Println("GAME OVER")
				game.gameState++
				game.displayBoard[TileX][TileY] = 12
				for H := range game.Width {
					for V := range game.Height {
						if game.proximityBoard[H][V] == 9 {
							if game.displayBoard[H][V] == 10 { game.displayBoard[H][V] = 9 }
						} else if game.displayBoard[H][V] == 11 {
							game.displayBoard[H][V] = 13
						}
					}
				}
			default:
				game.displayBoard[TileX][TileY] = game.proximityBoard[TileX][TileY]
			}
}

func (game *Game) Flag(TileX, TileY int) {
	if game.displayBoard[TileX][TileY] == 10 {
		game.displayBoard[TileX][TileY] = 11
		game.flagged++
	} else if game.displayBoard[TileX][TileY] == 11 {
		game.displayBoard[TileX][TileY] = 10
		game.flagged--
	}
	// I may want to make this some sort of channel?
	fmt.Printf("\x1b[2A\x1b[2KBombs: %d\x1b[2B\r", game.Bombs-game.flagged)
}

func (game *Game) Sweep(TileX, TileY int) {
	// Should probably have a "selectedtile" var here or smth.
	if game.proximityBoard[TileX][TileY] > 7 || game.proximityBoard[TileX][TileY] == 0 { return } //If it's less than 0, it should only be revealed as everything around it is cleared, so not needed. Obviously, uncovering a bomb ends the game. And if the tile is an 8, there's no point to counting up the tiles, since it won't have anywhere to dig.
		
	var targetFlagNum = game.proximityBoard[TileX][TileY]
	for _, Dropper := range game.inBoundsTilesAround(TileX, TileY) {
		if game.displayBoard[Dropper.X][Dropper.Y] == 11 { targetFlagNum-- }
	}
	if targetFlagNum == 0 {
		for _, dropper := range game.inBoundsTilesAround(TileX, TileY) {
			if game.displayBoard[dropper.X][dropper.Y] == 10 {
				game.Dig(dropper.X, dropper.Y)
			}
		}
	}
}

func (game *Game) exploreEmpty (Xclicked, Yclicked, Xfrom, Yfrom int) {
	var H, V int
	for _, dropper := range game.inBoundsTilesAround(Xclicked, Yclicked) {
		H, V = dropper.X, dropper.Y
		if game.displayBoard[H][V] == 10 {
			game.displayBoard[H][V] = game.proximityBoard[H][V]
			// Could try to insert the tile counter here, it depends on if this recounts itself.
			if game.proximityBoard[H][V] == 0 { game.exploreEmpty(H, V, Xclicked, Yclicked) } 
		}
	}
}

func (game *Game) Update() error {
	for _, A := range [4]*Action{&game.Digging, &game.Sweeping, &game.Flagging, &game.Resetting} { // Are pointers correct here?
		A.WasHeld = A.IsHeld
		A.IsHeld = false
		if A.UsesButtons {
			if ebiten.IsMouseButtonPressed(A.Buttons) { A.IsHeld = true }
		}
		if A.UsesKeys {
			if ebiten.IsKeyPressed(A.Keys) { A.IsHeld = true }
		}
	}

	if game.Resetting.IsHeld {
		if ebiten.IsKeyPressed(ebiten.Key1) { game.ResetTo = 1 }
		if ebiten.IsKeyPressed(ebiten.Key2) { game.ResetTo = 2 }
		if ebiten.IsKeyPressed(ebiten.Key3) { game.ResetTo = 3 }
		if ebiten.IsKeyPressed(ebiten.Key4) { game.ResetTo = 4 }
		// These can be action keys, but I just don't think it matters. The user is resetting anyways.
	}

	if !game.Resetting.IsHeld && game.Resetting.WasHeld {
		// RESET
		game.gameState = 0
		game.flagged = 0
		game.SetDifficulty(game.ResetTo)
		if game.ResetTo == 4 { game.ResetTo = 0 }
		game.iniDisplayBoard()
		game.ticker.Stop()
		fmt.Println("RESETTING GAME")
		fmt.Printf("Bombs: %d\nTime: 0\n", game.Bombs)
	}
	
	if game.gameState == 2 { return nil }

	if game.Digging.IsHeld && game.Flagging.IsHeld { // Chording. Should I have a special case where I look at the actual keys? What maps better in the brain lol.
		game.Digging.IsHeld = false ; game.Flagging.IsHeld = false
		game.Sweeping.IsHeld = true
	}
	// check if the above works
	// Also check for the other combos.
	if !game.Digging.IsHeld && game.Digging.WasHeld {
		if game.gameState == 0 {
			ClX, ClY := game.GetTiles()
			// I imagine there's a better way to do this
			game.iniGameBoards(ClX, ClY)
			game.firstClickTime = time.Now()
			game.gameState++
			// Start counting timing here
			game.ticker = time.NewTicker( time.Second )
			game.tickerDisplay = 1

			// Visual ticker
			go func() {
				for {
					<- game.ticker.C
					fmt.Printf("\x1b[1A\x1b[2KTime: %d\x1b[1B\r", game.tickerDisplay)
					game.tickerDisplay++
				}
			}()
		}
		// Get the hollow tiles
		game.Dig(game.GetTiles())
	}
	if !game.Sweeping.IsHeld && game.Sweeping.WasHeld { game.Sweep(game.GetTiles()) } // Hollow tiles
	if game.Flagging.IsHeld && !game.Flagging.WasHeld { game.Flag(game.GetTiles()) }

	
	var uncleared int
	for H := range game.Width {
		for V := range game.Height {
			if game.displayBoard[H][V] > 9 { uncleared++ }
		}
	}
	if uncleared == game.Bombs {
		fmt.Printf("\x1b[1A\x1b[2KTime: %.2f\x1b[1B\r", time.Since(game.firstClickTime).Seconds() )
		fmt.Println("3BV: ", game.BechtelValue())
		fmt.Println("YOU WIN")
		game.ticker.Stop()
		game.gameState++
	}
	// Ideally, I'd count up as tiles are cleared: it's probably more efficient than this.
	return nil
}

func (game *Game) Layout(RealWidth, RealHeight int) (LogicalWidth, LogicalHeight int){
	return game.TileSizeX * game.Width, game.TileSizeY * game.Height
}

type coord struct {
	X, Y int
}

func (game *Game) inBoundsTilesAround(Xclicked, Yclicked int) (Locations []coord) {
	// Is there a way to make this some sort of iterator?
	// Assumes the clicked tile is inbounds
	for H := range 3 {
		H += Xclicked - 1
		if (H < 0) || (H == game.Width) {continue}
		for V := range 3 {
			if (H == Xclicked) && (V == 1) {continue}
			V += Yclicked - 1
			if (V < 0) || (V == game.Height) {continue}
			Locations = append(Locations, coord{H, V})
		}
	}
	return Locations
}

func (game *Game) iniDisplayBoard() {
	game.displayBoard = make([][]int8, game.Width)
	for i := range game.Width {
		game.displayBoard[i] = make([]int8, game.Height)
		
		for Tiler := range game.Height {
			game.displayBoard[i][Tiler] = 10
		}
	}
	
	const ScaleConst int = 2
	ebiten.SetWindowSize(game.TileSizeX*game.Width*ScaleConst, game.TileSizeY*game.Height*ScaleConst) //to whatever real size we want it to display as.
}

func (game *Game) iniGameBoards (Xclicked, Yclicked int) {

	game.bombBoard = make([][]bool, game.Width)
	game.proximityBoard = make([][]int8, game.Width)
	
	for i := range game.Width {
		game.bombBoard[i] = make([]bool, game.Height)
		game.proximityBoard[i] = make([]int8, game.Height)
	}
	
	var BombPlace int = (Yclicked * game.Width) + Xclicked
	var BombLocation, BombLocX, BombLocY int
	var Spaces = game.Width * game.Height
	
	SRBombs := make([]int, Spaces)
	Spaces--
	for i := range Spaces { SRBombs[i] = i } // Can probably be done in one line, I don't know.

	SRBombs = append(SRBombs[:BombPlace], SRBombs[BombPlace+1:]...)
	
	for i := range game.Bombs {
		BombPlace = rand.IntN(Spaces - i)
		BombLocation = SRBombs[BombPlace]
		
		SRBombs = append( SRBombs[:BombPlace], SRBombs[BombPlace+1:]... )
		BombLocX, BombLocY = BombLocation % game.Width, BombLocation / game.Width
		game.bombBoard [ BombLocX ] [ BombLocY ] = true

		for _, validLocs := range game.inBoundsTilesAround(BombLocX, BombLocY) {
			if game.proximityBoard[validLocs.X][validLocs.Y] < 9 {
				game.proximityBoard[validLocs.X][validLocs.Y]++
			}
		}

		game.proximityBoard [ BombLocX ] [ BombLocY ] = 9

	}
}

func (game *Game) BechtelValue() (Clicks int) {
	// Implementation slightly inspired by:
	// https://gamedev.stackexchange.com/questions/63046/how-should-i-calculate-the-score-in-minesweeper-3bv-or-3bv-s
	var Cleared = make([][]bool, game.Width)
	for i := range game.Width {
		Cleared[i] = make([]bool, game.Height)
	}
	
	for V := range game.Height {
		for H := range game.Width {
			if Cleared[H][V] { continue }
			Cleared[H][V] = true
			switch game.proximityBoard[H][V] {
				case 9: continue // Bombs aren't counted, obviously
				case 0: // This is where I have to do the flood fill sweeping thing.
					aroundZero := false // I deviated from the implementation here, but I thought this was clever.
					for _, Surrounding := range game.inBoundsTilesAround(H, V) {
						if Cleared[Surrounding.X][Surrounding.Y] && game.proximityBoard[Surrounding.X][Surrounding.Y] == 0 {
							Cleared[H][V] = true
							aroundZero = true
							continue
						}
					}
					if !aroundZero { Clicks++ }
					// These are very similar, I should do something about that.
				default:
					aroundZero := false
					for _, i := range game.inBoundsTilesAround(H, V) {
						if game.proximityBoard[i.X][i.Y] == 0 { aroundZero = true ; continue }
					}
					if !aroundZero { Clicks++ }	
					
			}
		}			
	}
	return Clicks // Could (should?) do a naked return, but better for readability.
}

func (game *Game) SetDifficulty(Chosen int) {
	// 1..4: Beginner 8x8 10, Intermediate 16x16 40, Expert 30x16 99, Custom WxH B
	switch Chosen {
		case 1 : game.Width, game.Height, game.Bombs = 8, 8, 10
		case 2 : game.Width, game.Height, game.Bombs = 16, 16, 40
		case 3 : game.Width, game.Height, game.Bombs = 30, 16, 99
		case 4 :
			fmt.Print("In the format WxH B: ")
			fmt.Scanf("%dx%d %d", &game.Width, &game.Height, &game.Bombs)// Does this need a \n?
			// I should probably check for errors
			for game.Width <= 0 {
				fmt.Printf("Dimensions must be >= 0.\nWidth: ")
				fmt.Scanf("%d\n", &game.Width)
			}
			for game.Height <= 0 {
				fmt.Printf("Dimensions must be >= 0.\nHeight: ")
				fmt.Scanf("%d\n", &game.Height)
			}
			for game.Bombs >= game.Width * game.Height {
				fmt.Printf("Bombs must be less than Width * Height.\nBombs: ")
				fmt.Scanf("%d\n", &game.Bombs)
			}
			// if Bombs > (Width - 1) * (Height - 1) {fmt.Println("Warning: ")}
	}
}

func main() {

	game := &Game{}
	
	ebiten.SetScreenClearedEveryFrame(false)

	var Theme string = "ClassicXP"
	
	IndexFile, Error := os.Open(Theme + ".png") // Check if this is even openable/exists?
	defer IndexFile.Close() // I should check for if it fails to close.
	if Error != nil { fmt.Println(Error) }
	
	IndexImage, _, Error := image.Decode(IndexFile)
	if Error != nil { fmt.Println(Error) }
	game.index = ebiten.NewImageFromImage(IndexImage)

	game.TileSizeX, game.TileSizeY = game.index.Bounds().Max.X, game.index.Bounds().Max.Y
	if game.TileSizeY % 14 != 0 {
		fmt.Println("Malformed tileset error")
		//FIGURE OUT WAY TO END GAME
	} else {
		game.TileSizeY /= 14
	}

	
	// ebiten.KeyNumpad0 will be the "dummy" key: it still works, it's just there to display that KB keys can't be used.
	game.Digging = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton0,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	game.Sweeping = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton1,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	game.Flagging = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton2,
		UsesKeys: false,
		Keys: ebiten.KeyNumpad0,
	}
	game.Resetting = Action {
		UsesButtons: true,
		Buttons: ebiten.MouseButton4,
		UsesKeys: true,
		Keys: ebiten.KeyR,
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
		case 'B', 'b', '1' : game.SetDifficulty(1)
		case 'I', 'i', '2' : game.SetDifficulty(2)
		case 'E', 'e', '3' : game.SetDifficulty(3)
		case 'C', 'c', '4' : game.SetDifficulty(4)
		default:
			fmt.Println("Enter a valid difficulty, by # or (case insensitive) letter.")
			goto InvalidDifficultyEntered
	}
	fmt.Printf("Bombs: %d\nTime: 0\n", game.Bombs)

	// Difficulties:
	// Beginner - 8x8, 10
	// Intermediate - 16x16, 40
	// Expert - 30x16, 99
	// 3BV limits is 2, 30, 100

	game.iniDisplayBoard()
	// InitGameBoards runs on the first click; check in Update for when GameState == 0
	// Board is in (X, Y), starts at 0, 0 at top left.
	// Other work needs to know the location of the first click.
	
	game.ticker = time.NewTicker( time.Second )
	// This is so that if the user resets before they've clicked, there's a ticker to stop. Otherwise, I'm running .Stop() on something that doesn't exist.

	ebiten.SetWindowTitle("Minesweeper Clone")
	if Error = ebiten.RunGame( game ) ; Error != nil {
		fmt.Println(Error)
	}
}

func (game *Game)  Draw(Screen *ebiten.Image) {	
	for Hori := range game.Width {
		for Vert := range game.Height {
			game.Location.GeoM.Reset()
			game.Location.GeoM.Translate(float64(Hori*game.TileSizeX), float64(Vert*game.TileSizeY))
			tileIndex := int(game.displayBoard[Hori][Vert])

			// Hollow tiles
			// Put here because I don't want to have to track or modify the DisplayBoard
			// xt, yt := GetTiles()
			if game.Digging.IsHeld {
				if xt, yt := game.GetTiles() ; Hori == xt && Vert == yt {
					if tileIndex == 10 {
						tileIndex = 0
					}
				}
			}

			if game.Sweeping.IsHeld {
				for _, validLocs := range game.inBoundsTilesAround(game.GetTiles()) {
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
				image.Point{0, (game.TileSizeY * tileIndex)},
				image.Point{game.TileSizeX, (game.TileSizeY * (tileIndex + 1))}}
	
			//DisplayBoard[Hori][Vert] contains the "index" of tile we need to render.
			TileToDraw := game.index.SubImage(Rect).(*ebiten.Image)
			Screen.DrawImage(TileToDraw, &game.Location)
		}
	}
}
