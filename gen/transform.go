package gen

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	draw2 "golang.org/x/image/draw"

	"github.com/mccutchen/palettor"
	"github.com/treethought/impression-frame/util"
)

func applyPallate(img image.Image, pallet *palettor.Palette) (image.Image, error) {

	newRect := image.Rectangle{
		Min: img.Bounds().Min,
		Max: img.Bounds().Max,
	}
	finImage := image.NewRGBA(newRect)

	pColors := color.Palette(pallet.Colors())

	pixels := decodePixels(img, 0, 0)

	// slope := func(x, y int) int {
	// 	return (x * y) / 2
	// }

	for _, px := range pixels {
		cast := pColors.Convert(px.Color)
		// cast color to pallete, but making streaks of the same color across the new img

		// cast := pColors[casti]

		finImage.Set(
			px.Point.X,
			px.Point.Y,
			cast,
		)
	}
	return finImage, nil

}

func animate(img image.Image, outDir string) ([]image.Image, error) {
	pixels := decodePixels(img, 0, 0)

	imgs := []image.Image{}

	fmt.Println(len(pixels))
	for i := range pixels {
		if !(i%50000 == 0) {
			continue
		}
		fmt.Println(i)

		newRect := image.Rectangle{
			Min: img.Bounds().Min,
			Max: img.Bounds().Max,
		}
		newImg := image.NewRGBA(newRect)

		newPixs := decodePixels(img, i, 0)

		for idx, px := range newPixs {
			target := idx
			if (idx + 1) == len(pixels)-1 {
				target = 0
			}
			newImg.Set(
				target,
				px.Point.Y,
				px.Color,
			)
		}

		out := filepath.Join(outDir, fmt.Sprintf("%d.png", i))

		util.WriteImage(out, newImg)
		imgs = append(imgs, newImg)
	}
	return imgs, nil

}

func ShuffleImageColumns(img image.Image) (image.Image, error) {
	pixelCols, err := getPixelsByCol(img)
	if err != nil {
		log.Fatal(err)
	}
	newRect := image.Rectangle{
		Min: img.Bounds().Min,
		Max: img.Bounds().Max,
	}
	finImage := image.NewRGBA(newRect)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	idx := 0
	for _, i := range r.Perm(len(pixelCols)) {
		col := pixelCols[i]
		for _, px := range col {
			pi := px.Point.X
			if i%2 == 0 {
				pi = idx
			}

			finImage.Set(
				pi,
				px.Point.Y,
				px.Color,
			)
		}
		idx++

	}
	return finImage, nil
}

func ShuffleImageRows(img image.Image) (image.Image, error) {

	pixelRows, err := getPixels(img)
	if err != nil {
		log.Fatal(err)
	}

	newRect := image.Rectangle{
		Min: img.Bounds().Min,
		Max: img.Bounds().Max,
	}
	finImage := image.NewRGBA(newRect)

	r := rand.New(rand.NewSource(time.Now().Unix()))

	// idx := 0
	for idx, i := range r.Perm(len(pixelRows)) {
		row := pixelRows[i]
		for _, px := range row {

			pi := px.Point.Y
			if i%2 == 0 {
				pi = idx
			}
			finImage.Set(
				px.Point.X,
				pi,
				px.Color,
			)

		}
		idx++
	}

	draw.Draw(finImage, finImage.Bounds(), finImage, image.Point{0, 0}, draw.Src)
	return finImage, nil
}

func CombineImages(img1, img2 image.Image) image.Image {
	// collect pixel data from each image
	pixels1 := decodePixels(img1, 0, 0)
	pixels2 := decodePixels(img2, 0, 0)

	newRect := image.Rectangle{
		Min: img1.Bounds().Min,
		Max: img1.Bounds().Max,
	}
	finImage := image.NewRGBA(newRect)

	// mod := rand.Intn(2) + 1

	for i := range pixels1 {
		if i >= (len(pixels2) - 1) {
			continue

		}
		var px *Pixel
		// if i%2 == 0 || len(pixels2)-1 >= i {
		if i%2 == 0 {
			px = pixels1[i]
			finImage.Set(
				px.Point.X,
				px.Point.Y,
				px.Color,
			)
		} else {
			px = pixels2[i]
			finImage.Set(
				px.Point.X,
				px.Point.Y,
				px.Color,
			)
		}
	}
	draw.Draw(finImage, finImage.Bounds(), finImage, image.Point{0, 0}, draw.Src)
	return finImage
}


func embedText(img image.Image, text string) image.Image {
	// Get dimensions of the image
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Define the region in the middle of the image
	startX := width / 4
	endX := width * 3 / 4
	startY := height / 4
	endY := height * 3 / 4

	// Extract pixel data for the middle region
	pixels := getPixelsOfRegion(img, startX, endX, startY, endY)

	// Mapping pixel colors to characters based on palette
	// palette := getPalette(img)
	// palette := palettor.NewRGBPalette()
	// palette.AddColorsFromImage(img)

	// Example mapping: darker colors correspond to '#' and lighter colors correspond to '.'
	charMap := make(map[color.Color]rune)
	for y := 0; y < len(pixels); y++ {
		for x := 0; x < len(pixels[y]); x++ {
			_, _, _, a := pixels[y][x].Color.RGBA()
			// If alpha value is greater than 32768 (half of maximum), consider it as lighter color
			if a > 32768 {
				charMap[pixels[y][x].Color] = '.'
			} else {
				charMap[pixels[y][x].Color] = '#'
			}
			// You can add more conditions here to handle different color intensities if needed
		}
	}

	// Word to be formed using pixels
	word := "HELLO"
	// Rearrange pixels to spell the word
	charIndex := 0
	for y := 0; y < len(pixels); y++ {
		for x := 0; x < len(pixels[y]); x++ {
			if charIndex < len(word) {
				char := word[charIndex]
				if char != ' ' {
					pixels[y][x].Color = color.RGBA{0, 0, 0, 255} // replace with mapped color
					charIndex++
				}
				// Replace color with corresponding character from charMap
				// set pixel color to black
				pixels[y][x].Color = color.RGBA{0, 0, 0, 255}
				// pixels[y][x].Color = color.RGBA{0, 0, 0, 255} // for testing purpose, change it with charMap[pixels[y][x].Color]
				whiteC := color.RGBA{255, 255, 255, 255}
				if pixels[y][x].Color != whiteC {
					charIndex++
				}
			} else {
				break
			}
		}
	}

	// write the new pixels along with surrounding pixels to a new image
	newImg := image.NewRGBA(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if pointWithinRegion(x, y, startX, endX, startY, endY) {
				// make sure we have this indexes in the pixels array
				if y-startY < len(pixels) && x-startX < len(pixels[y-startY]) {
					newImg.Set(x, y, pixels[y-startY][x-startX].Color)
				}
			} else {
				newImg.Set(x, y, img.At(x, y))
			}
		}
	}

	// // Save the modified image
	// newImg := image.NewRGBA(bounds)
	// for y := startY; y < endY; y++ {
	// 	for x := startX; x < endX; x++ {
	// 		newImg.Set(x, y, pixels[y-startY][x-startX].Color)
	// 	}
	// }
	return newImg

}

func pointWithinRegion(px, py, x1, x2, y1, y2 int) bool {
	if px < x1 || px > x2 || py < y1 || py > y2 {
		return false
	}
	return px >= x1 && px <= x2 && py >= y1 && py <= y2
}

func writeImageWithinRegion(base image.Image, img image.Image, x1, x2, y1, y2 int) image.Image {
	// Get dimensions of the base image
	baseBounds := base.Bounds()
	baseWidth := baseBounds.Max.X
	baseHeight := baseBounds.Max.Y

	// Create a new RGBA image with the same dimensions as the base image
	finImage := image.NewRGBA(baseBounds)

	// Iterate over each pixel in the base image
	for y := 0; y < baseHeight; y++ {
		for x := 0; x < baseWidth; x++ {
			// Check if the current pixel is within the specified region
			if x >= x1 && x <= x2 && y >= y1 && y <= y2 {
				// If within the region, get the corresponding pixel from the img image
				finImage.Set(x, y, img.At(x-x1, y-y1))
			} else {
				// If outside the region, copy the pixel from the base image
				finImage.Set(x, y, base.At(x, y))
			}
		}
	}

	return finImage
}

func WriteWithin(base image.Image, img image.Image, scale int) image.Image {

	// scale is used to scale the image
	// we then use the scaled dimensions to write the image within the base image in the center

	scaled := scaleImage(img, scale)

	baseBounds := base.Bounds()
	scaledBounds := scaled.Bounds()

	x1 := (baseBounds.Max.X - scaledBounds.Max.X) / 2
	x2 := x1 + scaledBounds.Max.X
	y1 := (baseBounds.Max.Y - scaledBounds.Max.Y) / 2
	y2 := y1 + scaledBounds.Max.Y

	// write the scaled image within the base image
	result := writeImageWithinRegion(base, scaled, x1, x2, y1, y2)

	return result

}

func scaleImage(img image.Image, scale int) image.Image {
	// Calculate new dimensions
	width := int(float64(img.Bounds().Dx()) * float64(scale) / 100)
	height := int(float64(img.Bounds().Dy()) * float64(scale) / 100)

	// Create a new RGBA image with the new dimensions
	scaled := image.NewRGBA(image.Rect(0, 0, width, height))

	// Scale the image
	draw2.NearestNeighbor.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw2.Src, nil)
	// draw.FloydSteinberg.Draw(scaled, scaled.Bounds(), img, img.Bounds().Min)

	return scaled
}
