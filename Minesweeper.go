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
	if DisplayBoard[TileX][TileY] != 10 { return }
	switch ProximityBoard[TileX][TileY] {
			case 0:
				DisplayBoard[TileX][TileY] = 0
				ExploreEmpty(TileX, TileY, TileX, TileY)

			case 9:
				ticker.Stop()
				fmt.Println("GAME OVER")
				GameState++
				DisplayBoard[TileX][TileY] = 12
				for H := range Width {
					for V := range Height {
						if ProximityBoard[H][V] == 9 {
							if DisplayBoard[H][V] == 10 { DisplayBoard[H][V] = 9 }
						} else if DisplayBoard[H][V] == 11 {
							DisplayBoard[H][V] = 13
						}
					}
				}
			default:
				DisplayBoard[TileX][TileY] = ProximityBoard[TileX][TileY]
			}
}

func Flag(TileX, TileY int) {
	if DisplayBoard[TileX][TileY] == 10 {
		DisplayBoard[TileX][TileY] = 11
		Flagged++
	} else if DisplayBoard[TileX][TileY] == 11 {
		DisplayBoard[TileX][TileY] = 10
		Flagged--
	}
	// I may want to make this some sort of channel?
	fmt.Printf("\x1b[2A\x1b[2KBombs: %d\x1b[2B\r", Bombs-Flagged)
}

func Sweep(TileX, TileY int) {
	ValidLocs := InBoundsTilesAround(TileX, TileY)
	//Should probably have a "selectedtile" var here or smth.
	if ProximityBoard[TileX][TileY] > 7 || ProximityBoard[TileX][TileY] == 0 { return } //If it's less than 0, it should only be revealed as everything around it is cleared, so not needed. Obviously, uncovering a bomb ends the game. And if the tile is an 8, there's no point to counting up the tiles, since it won't have anywhere to dig.
		
	var TargetFlagNum = ProximityBoard[TileX][TileY]
	for Dropper := 0 ; Dropper < len(ValidLocs) ; Dropper +=2 {
		if DisplayBoard[ValidLocs[Dropper]][ValidLocs[Dropper+1]] == 11 { TargetFlagNum-- }
	}
	if TargetFlagNum == 0 {
		for Dropper := 0 ; Dropper < len(ValidLocs) ; Dropper += 2 {
			if DisplayBoard[ValidLocs[Dropper]][ValidLocs[Dropper+1]] == 10 {
				Dig(ValidLocs[Dropper], ValidLocs[Dropper+1])
			}
		}
	}
}

func ExploreEmpty (Xclicked, Yclicked, Xfrom, Yfrom int) {
	ValidLocs := InBoundsTilesAround(Xclicked, Yclicked)
	var H, V int
	for Dropper := 0 ; Dropper < len(ValidLocs) ; Dropper += 2 {
		H, V = ValidLocs[Dropper], ValidLocs[Dropper+1]
		if DisplayBoard[H][V] == 10 {
			DisplayBoard[H][V] = ProximityBoard[H][V]
			// Could try to insert the tile counter here, it depends on if this recounts itself.
			if ProximityBoard[H][V] == 0 { ExploreEmpty(H, V, Xclicked, Yclicked) } 
		}
	}
}


var (
	Flagged int
	GameState int // 0 is on first click, 1 is actively playing, 2 is over
	ResetTo int
	ticker *time.Ticker
	tickerDisplay int
)

type Action struct {
	isHeld bool
	wasHeld bool // Last tick
	usesButtons bool
	usesKeys bool
	buttons ebiten.MouseButton
	keys ebiten.Key
}
// I'm sure something clever could be done with channels

var(
	Digging = Action {
		usesButtons: true,
		buttons: ebiten.MouseButton0,
		usesKeys: false,
		keys: ebiten.KeyNumpad0,
	}
	Sweeping = Action {
		usesButtons: true,
		buttons: ebiten.MouseButton1,
		usesKeys: false,
		keys: ebiten.KeyNumpad0,
	}
	Flagging = Action {
		usesButtons: true,
		buttons: ebiten.MouseButton2,
		usesKeys: false,
		keys: ebiten.KeyNumpad0,
	}
	Resetting = Action {
		usesButtons: true,
		buttons: ebiten.MouseButton4,
		usesKeys: true,
		keys: ebiten.KeyR,
	}
	// Should I bother making a quitting action?
	// Probably :(
)

func (Minefield *Game) Update() error {
	// ebiten.KeyNumpad0 will be the "dummy" key: it still works, it's just there to display that KB keys can't be used.
	for _, A := range [4]*Action{&Digging, &Sweeping, &Flagging, &Resetting} { // Are pointers correct here?
		A.wasHeld = A.isHeld
		A.isHeld = false
		if A.usesButtons {
			if ebiten.IsMouseButtonPressed(A.buttons) { A.isHeld = true }
		}
		if A.usesKeys {
			if ebiten.IsKeyPressed(A.keys) { A.isHeld = true }
		}
	}

	if Resetting.isHeld {
		if ebiten.IsKeyPressed(ebiten.Key1) { ResetTo = 1 }
		if ebiten.IsKeyPressed(ebiten.Key2) { ResetTo = 2 }
		if ebiten.IsKeyPressed(ebiten.Key3) { ResetTo = 3 }
		if ebiten.IsKeyPressed(ebiten.Key4) { ResetTo = 4 }
		// These can be action keys, but I just don't think it matters. The user is resetting anyways.
	}

	if !Resetting.isHeld && Resetting.wasHeld {
		// RESET
		GameState = 0
		Flagged = 0
		SetDifficulty(ResetTo)
		if ResetTo == 4 { ResetTo = 0 }
		DisplayBoard = IniDisplayBoard(Width, Height)
		ticker.Stop()	
		fmt.Println("RESETTING GAME")
		fmt.Printf("Bombs: %d\nTime: 0\n", Bombs)
	}
	
	if GameState == 2 { return nil }

	if Digging.isHeld && Flagging.isHeld { // Chording. Should I have a special case where I look at the actual keys? What maps better in the brain lol.
		Digging.isHeld = false ; Flagging.isHeld = false
		Sweeping.isHeld = true
	}
	// check if the above works
	// Also check for the other combos.
	if !Digging.isHeld && Digging.wasHeld {
		if GameState == 0 {
			ClX, ClY := GetTiles()
			// I imagine there's a better way to do this
			ProximityBoard, BombBoard = IniGameBoards(ClX, ClY, Width, Height, Bombs)
			GameState++
			BechtelValue()
			// Start counting timing here
			ticker = time.NewTicker( time.Second )
			tickerDisplay = 1

			// Visual ticker
			go func() {
				for {
					// For some reason, always sends one extra tick after it's been stopped.
					select {
						case <- ticker.C:
							fmt.Printf("\x1b[1A\x1b[2KTime: %d\x1b[1B\r", tickerDisplay)
							tickerDisplay++
					}
				}
			} ()
		}
		// Get the hollow tiles
		Dig(GetTiles())
	}
	if !Sweeping.isHeld && Sweeping.wasHeld { Sweep(GetTiles()) } // Hollow tiles
	if Flagging.isHeld && !Flagging.wasHeld { Flag(GetTiles()) }

	
	var Uncleared int
	for H := range Width {
		for V := range Height {
			if DisplayBoard[H][V] > 9 { Uncleared++ }
		}
	}
	if Uncleared == Bombs {
		fmt.Println("YOU WIN")
		ticker.Stop()
		GameState++
	}
	// Ideally, I'd count up as tiles are cleared: it's probably more efficient than this.
	// I hate most
	return nil
}

func (Minefield *Game) Layout(RealWidth, RealHeight int) (LogicalWidth, LogicalHeight int){
	return TileSizeX * Width, TileSizeY * Height
}

var(
Height, Width, Bombs, TileSizeX, TileSizeY int
ProximityBoard, DisplayBoard [][]int8
Error error
Index *ebiten.Image
Location ebiten.DrawImageOptions
BombBoard [][]bool
)
func InBoundsTilesAround(Xclicked, Yclicked int) (Locations []int) {
	// Is there a way to make this some sort of iterator?
	// Assumes the clicked tile is inbounds
	for H := range 3 {
		H += Xclicked - 1
		if (H < 0) || (H == Width) {continue}
		for V := range 3 {
			if (H == Xclicked) && (V == 1) {continue}
			// Slightly confusing, but it removes the clicked tile from being included, and removes a VERY small amount of processing.
			V += Yclicked - 1
			if (V < 0) || (V == Height) {continue}
			Locations = append(Locations, H, V)
		}
	}
	return
}

func IniDisplayBoard (Width, Height int) (DisplayBoard [][]int8) {
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

func IniGameBoards (Xclicked, Yclicked, Width, Height, Bombs int) (ProximityBoard [][]int8, BombBoard [][]bool) {

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

		ValidLocs := InBoundsTilesAround(BombLocX, BombLocY)
		for Dropper := 0 ; Dropper < len(ValidLocs) ; Dropper += 2 {
			if ProximityBoard[ValidLocs[Dropper]][ValidLocs[Dropper+1]] < 9 {
				ProximityBoard[ValidLocs[Dropper]][ValidLocs[Dropper+1]]++
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
			switch ProximityBoard[H][V] {
				case 9: continue // Bombs aren't counted, obviously
				case 0: // This is where I have to do the flood fill sweeping thing.
					aroundZero := false // I deviated from the implementation here, but I thought this was clever.
					Surrounding := InBoundsTilesAround(H, V)
					for i:= 0 ; i < len(Surrounding) ; i += 2 {
						tX, tY := Surrounding[i], Surrounding[i+1]
						if Cleared[tX][tY] && ProximityBoard[tX][tY] == 0 {
							Cleared[H][V] = true
							aroundZero = true
							continue
						}
					}
					if !aroundZero { Clicks++ }
					// These are very similar, I should do something about that.
				default:
					aroundZero := false
					Surrounding := InBoundsTilesAround(H, V)
					for i := 0 ; i < len(Surrounding) ; i += 2 {
						if ProximityBoard[Surrounding[i]][Surrounding[i+1]] == 0 { aroundZero = true ; continue }
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
			for Bombs >= Width * Height {
				fmt.Println("Bombs must be less than Width * Height.")
				fmt.Print("Bombs: ") ; fmt.Scanf("%d\n", &Bombs)
			}
			// if Bombs > (Width - 1) * (Height - 1) {fmt.Println("Warning: ")}
	}
}

// TODO
// Time and allat
	
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
	_, Error = fmt.Scanf("%c\n", &Difficulty)
	// Sterilize this
	if Error != nil { fmt.Println(Error) }
	switch Difficulty {
		case 'B', 'b', '1' : SetDifficulty(1)
		case 'I', 'i', '2' : SetDifficulty(2)
		case 'E', 'e', '3' : SetDifficulty(3)
		case 'C', 'c', '4' : SetDifficulty(4)
	}
	fmt.Printf("Bombs: %d\nTime: 0\n", Bombs)

	// Difficulties:
	// Beginner - 8x8, 10
	// Intermediate - 16x16, 40
	// Expert - 30x16, 99
	// 3BV limits is 2, 30, 100

	DisplayBoard = IniDisplayBoard(Width, Height)
	// InitGameBoards runs on the first click; check in Update for when GameState == 0
	// Board is in (X, Y), starts at 0, 0 at top left.
	// Other work needs to know the location of the first click.

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
			TileIndex := int(DisplayBoard[Hori][Vert])
			Rect := image.Rectangle{
				image.Point{0, (TileSizeY * TileIndex)},
				image.Point{TileSizeX, (TileSizeY * (TileIndex + 1))}}
	
			//DisplayBoard[Hori][Vert] contains the "index" of tile we need to render.
			TileToDraw := Index.SubImage(Rect).(*ebiten.Image)
			Screen.DrawImage(TileToDraw, &Location)
		}
	}
}
