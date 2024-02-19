package gen

import (
	"image"
	"image/color"
)

// Create a struct to deal with pixel
type Pixel struct {
	Point image.Point
	Color color.Color
}

// Decode image.Image's pixel data into []*Pixel
func decodePixels(img image.Image, offsetX, offsetY int) []*Pixel {
	pixels := []*Pixel{}
	for y := 0; y <= img.Bounds().Max.Y; y++ {
		for x := 0; x <= img.Bounds().Max.X; x++ {
			p := &Pixel{
				Point: image.Point{x + offsetX, y + offsetY},
				Color: img.At(x, y),
			}
			pixels = append(pixels, p)
		}
	}
	return pixels
}

// Get the bi-dimensional pixel array
func getPixels(img image.Image) ([][]Pixel, error) {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]Pixel
	for y := 0; y < height; y++ {
		var row []Pixel
		for x := 0; x < width; x++ {
			p := Pixel{
				Point: image.Point{x, y},
				Color: img.At(x, y),
			}
			row = append(row, p)
		}
		pixels = append(pixels, row)
	}

	return pixels, nil
}

func getPixelsByCol(img image.Image) ([][]Pixel, error) {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]Pixel
	for x := 0; x < width; x++ {
		var col []Pixel
		for y := 0; y < height; y++ {
			p := Pixel{
				Point: image.Point{x, y},
				Color: img.At(x, y),
			}
			col = append(col, p)
		}
		pixels = append(pixels, col)
	}
	return pixels, nil

}

func getPixelsOfRegion(img image.Image, x1, x2, y1, y2 int) [][]Pixel {
  var pixels [][]Pixel
  for y := y1; y < y2; y++ {
    var row []Pixel
    for x := x1; x < x2; x++ {
      p := Pixel{
        Point: image.Point{x, y},
        Color: img.At(x, y),
      }
      row = append(row, p)
    }
    pixels = append(pixels, row)
  }
  return pixels
}

// shufflePixels randomly shuffles the pixels of the inner slice of pixels
// func shufflePixels(pixels [][]Pixel, toImg *image.RGBA) (error) {
// 	r := rand.New(rand.NewSource(time.Now().Unix()))

// 	idx := 0
// 	for _, i := range r.Perm(len(pixels)) {
// 		slice := pixels[i]
// 		for _, px := range slice {
// 			finImage.Set(
// 				px.Point.X,
// 				idx,
// 				px.Color,
// 			)

// 		}
// 		idx++
// 	}

// }
