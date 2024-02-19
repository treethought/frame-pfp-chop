package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	dither "github.com/makeworld-the-better-one/dither/v2"
	"github.com/mccutchen/palettor"
)

func ditherImg(img image.Image, pallet *palettor.Palette) image.Image {

	d := dither.NewDitherer(pallet.Colors())

	d.Matrix = dither.FloydSteinberg
	d.Matrix = dither.JarvisJudiceNinke

	return d.DitherCopy(img)
}

func createMovingGif() {
	img, _, err := loadRandomUnsplashImage(2000, 2000)
	if err != nil {
		log.Fatal(err)
	}
	_, err = animate(img, "animate-stage")
	if err != nil {
		log.Fatal(err)
	}

	imgs := []image.Image{}
	entries, err := os.ReadDir("animate-stage")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		img, _, err := loadImage(filepath.Join("animate-stage", e.Name()))
		if err != nil {
			log.Fatal(err)
		}
		imgs = append(imgs, img)
	}
	createGif("animate.gif", false, imgs...)

	return

}

func runPalleteMix() {
	outDir := "./out2023"
	runner := newRunner(outDir)
	runner.transforms = make(map[string]transformer)
	runner.transforms["horizontal"] = shuffleImageColumns
	runner.transforms["vertical"] = shuffleImageRows
	imgs := []image.Image{}

	urls := make(map[string]image.Image)
	urlset := []string{}

	// topics := []string{"flowers", "ocean", "sunset", "architecture", "landscape", "nature"}
	topics := []string{"autumn", "spring", "garden", "summer", "lights"}

	useAPI := true

	if !useAPI {
		i := 0
		for i < 11 {
			uimag, url, err := loadRandomUnsplashImage(2000, 2000)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("checkin ", url)
			if _, ok := urls[url]; ok {
				fmt.Println("udp, retrying in 4 seconds")
				time.Sleep(4)
				continue
			}
			fmt.Println(url)
			urls[url] = uimag
			urlset = append(urlset, url)
			i++

		}
	} else {

		i := 0
		for i < 5 {
			topic := topics[rand.Intn(len(topics))]
			rimages, _, err := getRandomImages(2000, 2000, topic, 3)
			if err != nil {
				log.Println("failed to get image: ", err)
			}
			imgs = append(imgs, rimages...)
			i++
		}

	}

	var wg1 sync.WaitGroup

	runImages := []image.Image{}
	if useAPI {
		runImages = imgs
	} else {
		for _, dimg := range urls {
			runImages = append(runImages, dimg)
		}

	}

	for _, dimg := range runImages {
		wg1.Add(1)
		go func(dimg image.Image) {

			uidx := rand.Intn(len(runImages))
			// img2 := urls[urlset[uidx]]
			img2 := runImages[uidx]

			p1 := getPalette(dimg)
			p2 := getPalette(img2)
			cimg := combineImages(dimg, img2)

			runner.palletes = append(runner.palletes, p1, p2)
			runImages = append(runImages, cimg)
			wg1.Done()

			// runner.run(img)
		}(dimg)
	}
	wg1.Wait()

	var wg sync.WaitGroup
	for _, img := range runImages {
		wg.Add(1)
		go func(img image.Image) {
			runner.mixPallete(img)
			wg.Done()
		}(img)
	}
	wg.Wait()

	runner.writeReuslts()
}

func runShuffles() {
	outDir := "./out2023"
	i := 0
	runner := newRunner(outDir)
	runner.transforms = make(map[string]transformer)
	runner.transforms["horizontal"] = shuffleImageColumns
	runner.transforms["vertical"] = shuffleImageRows
	palletes := []*palettor.Palette{}

	for i < 11 {
		// go func() {
		var img image.Image
		var err error

		if len(os.Args) == 1 {
			img, _, err = loadRandomUnsplashImage(2000, 2000)
			if err != nil {
				log.Fatal(err)
			}

			p := getPalette(img)
			palletes = append(palletes, p)

		} else {
			f, err := os.Open(os.Args[1])
			if err != nil {
				log.Fatal(err)
			}

			img, _, err = image.Decode(f)
			if err != nil {
				log.Fatal(err)
			}
		}

		runner.run(img)
		i++
	}
	runner.writeReuslts()
	cid := runner.uploadResults()

	runner.generateMeta(cid)
	runner.uploadMeta()

}
