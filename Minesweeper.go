package main

import(
	"fmt"
	"math/rand/v2"
	"github.com/hajimehoshi/ebiten/v2"
	//"image"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct{}

func GetTiles() (TileX, TileY int){
	ClickedPosX, ClickedPosY := ebiten.CursorPosition()		
	TileX = ClickedPosX / TileSize
	TileY = ClickedPosY / TileSize
	return
}

func (Minefield *Game) Update() error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		MouseLeftHeld = true
	} else {
		if MouseLeftHeld {
			MouseLeftHeld = false
			TileX, TileY := GetTiles()		
			switch ProximityBoard[TileX][TileY] {
			case 0:
				DisplayBoard[TileX][TileY] = 0
				ExploreEmpty(TileX, TileY, TileX, TileY)

			case 9:
				fmt.Println("GAME OVER")
				DisplayBoard[TileX][TileY] = 12
				for H := range Width {
					for V := range Height {
						if ProximityBoard[H][V] == 9 {
							if DisplayBoard[H][V] == 11 { DisplayBoard[H][V] = 9 }
						} else if DisplayBoard[H][V] == 10 {
							DisplayBoard[H][V] = 13
						}
					}
				}
			default:
				DisplayBoard[TileX][TileY] = ProximityBoard[TileX][TileY]
			}
		}
	}
	
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if !MouseRightHeld {
			TileX, TileY := GetTiles()
			if DisplayBoard[TileX][TileY] >= 10 {
				DisplayBoard[TileX][TileY] ^= 1
			}
		}
		MouseRightHeld = true
	} else {
		MouseRightHeld = false
	}
	return nil
}

func (Minefield *Game) Layout(RealWidth, RealHeight int) (LogicalWidth, LogicalHeight int){
	return TileSize * Width, TileSize * Height
}

func ImgFromPath (TileIn string) (ImageOut *ebiten.Image) {
	//Might want to add support for non-png images.
	var FilePath string = Theme + "/" + TileIn + ".png"
	ImageOut, _, Error := ebitenutil.NewImageFromFile(FilePath)
	if Error != nil {
		fmt.Println(Error)
	} 
	return
}

var(
Height, Width, Bombs, Spaces, TileSize int
ProximityBoard, DisplayBoard [][]int8
Theme string = "beetrootpaul"
Tiles map[int8]*ebiten.Image
MouseLeftHeld, MouseRightHeld bool
)

func InBounds (X, Y int) (In bool) {
	In = true
	if X < 0 { In = false }
	if X >= Width { In = false }
	if Y < 0 { In = false }
	if Y >= Height { In = false }
	return
}

func ExploreEmpty (Xclicked, Yclicked, Xfrom, Yfrom int) {
	for _, h := range [3]int{-1, 0, 1} {
		H := Xclicked + h
		for _, v := range [3]int{-1, 0, 1} {
			V := Yclicked + v
			
			if InBounds(H, V) {
				if !((H==Xclicked && V==Yclicked) || (H==Xfrom && V==Yfrom)) {
					if DisplayBoard[H][V] == 11 {
						if ProximityBoard[H][V] == 0 {
							DisplayBoard[H][V] = 0
							ExploreEmpty(H, V, Xclicked, Yclicked)
						} else {
							DisplayBoard[H][V] = ProximityBoard[H][V]
						}
					}
				}
			}
		}
	}
}
	
func main() {
	ebiten.SetScreenClearedEveryFrame(false)
	Tiles = make(map[int8]*ebiten.Image)
	for i := range int8(9) {
		Tiles[i] = ImgFromPath(fmt.Sprint(i))
	}

	Tiles[9] = ImgFromPath("bomb")
	Tiles[10] = ImgFromPath("flag")
	Tiles[11] = ImgFromPath("tile")
	Tiles[12] = ImgFromPath("hit")
	Tiles[13] = ImgFromPath("wrong")

	//Rect := image.Rectangle{
	//	image.Point{0, 0},
	//	image.Point{15, 15}}
	
	//Index := ImgFromPath("index")
	//for i := range 14 {
	//	fmt.Println(i)
	//	Tiles[int8(i)] = ebiten.NewImageFromImage(
	//		Index.SubImage(Rect))
	//	Rect = Rect.Add(image.Point{0, 16})
	//	fmt.Println(Rect)
	//}	

	
	Height = 5
	Width = 5
	Bombs = 5 // Cannot be more than Spaces
	Spaces = Height * Width
	TileSize = 16
	BombBoard := make([][]bool, Width)
	ProximityBoard, DisplayBoard = make([][]int8, Width), make([][]int8, Width)
	for i := range Width {
		BombBoard[i] = make([]bool, Height)
		ProximityBoard[i], DisplayBoard[i] = make([]int8, Height), make([]int8, Height)
		for Tiler := range Height {
			DisplayBoard[i][Tiler] = 11
		}
	}//Board is in (X, Y), starts at 0, 0 at top left.
	
	var SRBombs []int = make([]int, Spaces, Spaces)
	for i := range Spaces { SRBombs[i] = i } // Can probably be done in one line, I don't know.
	
	for i := range Bombs {
		var BombPlace int = rand.IntN(Spaces - i)
		var BombLocation = SRBombs[BombPlace]
		fmt.Println(BombLocation)
		SRBombs = append( SRBombs[:BombPlace], SRBombs[BombPlace+1:]... )
		var BombLocX, BombLocY = BombLocation % Width, BombLocation / Height
		BombBoard [ BombLocX ] [ BombLocY ] = true

		for _, LTR := range [3]int{-1, 0, 1} {
			for _, UTD := range [3]int{-1, 0, 1} {
				if InBounds(BombLocX + LTR, BombLocY + UTD) {
					if ProximityBoard[BombLocX + LTR][BombLocY + UTD] < 9 {					ProximityBoard[BombLocX + LTR][BombLocY + UTD]++
					}
				}
			}
		}

		ProximityBoard [ BombLocX ] [ BombLocY ] = 9
		
	}

	fmt.Println(ProximityBoard)

	for Y := range Height {
		for X := range Width {
			fmt.Print(ProximityBoard[X][Y], ", ")
		}
		fmt.Println()
	}


	ebiten.SetWindowSize(320, 320) //to whatever real size we want it to display as.
	ebiten.SetWindowTitle("Minesweeper Clone")
	if Error := ebiten.RunGame( &Game{} ) ; Error != nil {
		fmt.Println(Error)
	}
}

func (Minefield *Game)  Draw(Screen *ebiten.Image) {	
	Location := &ebiten.DrawImageOptions{}
	Location.GeoM.Translate(float64(-TileSize), float64( TileSize * (Height - 1) ))
	for Hori := range Width {
		Location.GeoM.Translate(float64(TileSize), float64(-TileSize * Height))
		for Vert := range Height {
			Location.GeoM.Translate(0, float64(TileSize))
			Screen.DrawImage(Tiles[DisplayBoard[Hori][Vert]], Location)
		}
	}
}