package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/andybons/gogif"
	"github.com/mccutchen/palettor"
	"github.com/nfnt/resize"
	"github.com/phrozen/blend"
)

var client = http.Client{}

func fetchImage(url string) (image.Image, string, error) {
	log.Println("fetching image: ", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	img, format, err := image.Decode(res.Body)
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

func loadImage(filepath string) (image.Image, string, error) {
	imgFile, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer imgFile.Close()
	img, format, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}
	return img, format, nil
}

type Response struct {
	Urls map[string]string
}

func getRandomImages(width, height int, query string, count int) ([]image.Image, string, error) {
	images := []image.Image{}

	key := "EFBmjsTwzleb3SAdc_-VSS263LyE2dIs2hYscFbqdds"
	url := fmt.Sprintf("https://api.unsplash.com/photos/random?client_id=%s&query=%s&h=%d&w=%d&count=%d", key, query, height, width, count)
	res, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	bdy, err := io.ReadAll(res.Body)

	data := []Response{}

	err = json.Unmarshal(bdy, &data)
	if err != nil {
		log.Println("fialed to unmarshal")
		log.Println("status: ", res.Status)
		log.Println("body: ", string(bdy))
		log.Fatal(err)
	}

	for _, im := range data {

		imgUrl := fmt.Sprintf("%s?w=%d&h=%d&client_id=%s", im.Urls["full"], width, height, key)
		fmt.Println(imgUrl)

		req, err := http.NewRequest(http.MethodGet, imgUrl, nil)
		req.Header.Set("Authorization", fmt.Sprintf("Client-ID %s", key))
		imgRes, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}
		fmt.Println(imgRes.Status)

		img, _, err := image.Decode(imgRes.Body)
		if err != nil {
			return nil, "", err
		}
		images = append(images, img)

	}

	return images, "", err
}

func getRandomImage(width, height int, query string) (image.Image, string, error) {

	key := "EFBmjsTwzleb3SAdc_-VSS263LyE2dIs2hYscFbqdds"
	url := fmt.Sprintf("https://api.unsplash.com/photos/random?client_id=%s&h=%d&w=%d", key, height, width)
	res, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	bdy, err := io.ReadAll(res.Body)

	data := Response{}

	err = json.Unmarshal(bdy, &data)
	if err != nil {
		log.Println("fialed to unmarshal")
		log.Println("status: ", res.Status)
		log.Println("body: ", string(bdy))
		log.Fatal(err)
	}

	imgUrl := fmt.Sprintf("%s?w=%d&h=%d&client_id=%s", data.Urls["full"], width, height, key)
	fmt.Println(imgUrl)

	req, err := http.NewRequest(http.MethodGet, imgUrl, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Client-ID %s", key))
	imgRes, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	fmt.Println(imgRes.Status)

	img, _, err := image.Decode(imgRes.Body)
	if err != nil {
		return nil, "", err
	}

	return img, imgUrl, err
}

func loadRandomUnsplashImage(width, height int) (image.Image, string, error) {
	url := fmt.Sprintf("https://source.unsplash.com/random/%dx%d", width, height)
	res, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	imgUrl := res.Request.URL.String()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, "", err
	}

	return img, imgUrl, err
}

func writeImage(path string, img image.Image) {
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	if filepath.Ext(path) == ".png" {
		err = png.Encode(out, img)
	} else {
		err = jpeg.Encode(out, img, nil)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func fillBlend(imgs ...image.Image) []image.Image {
	fmt.Println("blending...")
	s := []image.Image{}
	for idx, img := range imgs {
		if idx == 0 {
			s = append(s, img)
			continue
		}
		i1 := imgs[idx-1]
		// dst := i1.(draw.Image)
		// blend.BlendImage(dst, img, blend.ColorDodge)
		blended := blend.BlendNewImage(i1, img, blend.ColorDodge)
		// blended := blend.BlendNewImage(i1, img, blend.Luminosity)
		// blended := blend.BlendNewImage(i1, img, blend.Multiply)
		s = append(s, blended, img)
	}
	return s
}

func createGif(out string, blend bool, imgs ...image.Image) {
	fmt.Println("creating gif")
	// save to out.gif
	f, _ := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()

	// if blend {
	// 	imgs = fillBlend(imgs...)
	// }

	var wg sync.WaitGroup
	wg.Add(len(imgs))

	quantizer := gogif.MedianCutQuantizer{NumColor: 64}
	// load static image and construct outGif
	outGif := &gif.GIF{}
	for i, img := range imgs {
		go func(idx int, img image.Image) {
			fmt.Print(".")
			// https://stackoverflow.com/questions/35850753/how-to-convert-image-rgba-image-image-to-image-paletted
			bounds := img.Bounds()
			palettedImage := image.NewPaletted(bounds, nil)
			quantizer.Quantize(palettedImage, bounds, img, image.ZP)
			// Add new frame to animated GIF
			outGif.Image = append(outGif.Image, palettedImage)
			// blended frame
			// if blend && idx%2 != 0 {
			// 	delay = 10
			// }

			outGif.Delay = append(outGif.Delay, 10)
			fmt.Print("➰")
			wg.Done()
		}(i, img)
	}
	wg.Wait()
	fmt.Println("\n", out)

	gif.EncodeAll(f, outGif)
}

func getPalette(img image.Image) *palettor.Palette {
	if img == nil {
		return nil
	}

	// Reduce it to a manageable size
	thumb := resize.Thumbnail(200, 200, img, resize.Lanczos3)

	// Extract the 3 most dominant colors, halting the clustering algorithm
	// after 100 iterations if the clusters have not yet converged.
	k := 4
	maxIterations := 1000
	palette, err := palettor.Extract(k, maxIterations, thumb)

	// Err will only be non-nil if k is larger than the number of pixels in the
	// input image.
	if err != nil {
		log.Fatalf("image too small")
	}

	// Palette is a mapping from color to the weight of that color's cluster,
	// which can be used as an approximation for that color's relative
	// dominance
	for _, color := range palette.Colors() {
		log.Printf("color: %v; weight: %v", color, palette.Weight(color))
	}
	return palette
	// drawPalette(os.Stdout, thumb, palette, "jpeg")

}

// Draw a palette over the bottom 10% of an image
func drawPalette(dst io.Writer, img image.Image, palette *palettor.Palette, format string) error {

	// Ensure we're working with RGBA data, which is necessary to a) have more
	// immediately useful JSON output and b) allow us to draw a palette back
	// onto the source image.
	//
	// In particular, JPEGs decode to *image.YCbCr, which must be converted to
	// *image.RGBA before we can draw our palette onto it.
	//
	// https://stackoverflow.com/a/47539710/151221
	if _, ok := img.(*image.RGBA); !ok {
		img2 := image.NewRGBA(img.Bounds())
		draw.Draw(img2, img2.Bounds(), img, image.Point{}, draw.Src)
		img = img2
	}

	drawImg := img.(draw.Image)

	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	paletteHeight := int(math.Ceil(float64(imgHeight) * 0.1))
	yOffset := imgHeight - paletteHeight
	xOffset := 0

	for _, entry := range palette.Entries() {
		colorWidth := int(math.Ceil(float64(imgWidth) * entry.Weight))
		bounds := image.Rect(xOffset, yOffset, xOffset+colorWidth, yOffset+paletteHeight)
		draw.Draw(drawImg, bounds, &image.Uniform{entry.Color}, image.Point{}, draw.Src)
		xOffset += colorWidth
	}

	switch format {
	case "jpeg":
		return jpeg.Encode(dst, drawImg, nil)
	case "gif":
		return gif.Encode(dst, drawImg, nil)
	default:
		return png.Encode(dst, drawImg)
	}
}

func hashImage(img image.Image) string {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		log.Fatal(err)
	}
	imgBytes := buf.Bytes()
	h := sha256.Sum256(imgBytes)
	return fmt.Sprintf("%x", h)
}
