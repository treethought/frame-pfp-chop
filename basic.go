package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func runImage(id string, img image.Image) {
	outDir := filepath.Join("./results", id)

	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	ogFile := filepath.Join(outDir, "original.png")
	writeImage(ogFile, img)

	imgs := []image.Image{}

	// shuffle rows
	shRow, err := shuffleImageRows(img)
	if err != nil {
		log.Fatal(err)
	}

	rowFile := filepath.Join(outDir, "shuffle-row.png")
	writeImage(rowFile, shRow)
	imgs = append(imgs, shRow)

	// shuffle columns
	shCol, err := shuffleImageColumns(img)
	if err != nil {
		log.Fatal(err)
	}

	colFile := filepath.Join(outDir, "shuffle-col.png")
	writeImage(colFile, shCol)
	imgs = append(imgs, shCol)

	// shuffle columns of shuffled rows
	cross, err := shuffleImageColumns(shRow)
	crossFile := filepath.Join(outDir, "shuffle-cross.png")
	writeImage(crossFile, cross)
	imgs = append(imgs, cross)

	// combine the two shuffles
	shCombine := combineImages(shRow, shCol)
	combineFile := filepath.Join(outDir, "shuffle-combine.png")
	writeImage(combineFile, shCombine)
	imgs = append(imgs, shCombine)

	// combine shuffled cols and og
	combineCols := combineImages(img, shCol)
	combineColsFile := filepath.Join(outDir, "result-cols.png")
	writeImage(combineColsFile, combineCols)
	// imgs = append(imgs, combineCols)

	// combine shuffled rows and og
	combineRows := combineImages(img, shRow)
	combineRowsFile := filepath.Join(outDir, "result-rows.png")
	writeImage(combineRowsFile, combineRows)
	// imgs = append(imgs, combineRows)

	// combine the combined
	finImage := combineImages(combineRows, combineCols)
	resultFile := filepath.Join(outDir, "result-combine.png")
	writeImage(resultFile, finImage)
	// imgs = append(imgs, finImage)

	gifFile := filepath.Join(outDir, "result.gif")
	createGif(gifFile, true, imgs...)

	fmt.Println("results:", outDir)

	c := exec.Command("open", ogFile, rowFile, colFile, crossFile, combineFile, combineColsFile, combineRowsFile, resultFile)
	c.Run()
}

func runImageCycle(id string, img image.Image) {
	fmt.Println("forming impression...")

	outDir := filepath.Join("./results", id)

	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	ogFile := filepath.Join(outDir, "original.png")
	writeImage(ogFile, img)

	imgs := []image.Image{}

	var wg sync.WaitGroup

	size := 5

	i := 0
	wg.Add(size)
	for i < size {
		i++
		go func(img image.Image) {
			fmt.Print(".")
			// shuffle rows
			fmt.Print("s")
			shRow, err := shuffleImageRows(img)
			if err != nil {
				log.Fatal(err)
			}
			imgs = append(imgs, shRow)

			fmt.Print("c")
			shCol, err := shuffleImageColumns(img)
			if err != nil {
				log.Fatal(err)
			}
			imgs = append(imgs, shCol)

			fmt.Print("x")
			// shuffle columns of shuffled rows
			cross, err := shuffleImageColumns(shRow)
			imgs = append(imgs, cross)

			fmt.Print("❀")
			// combine the two shuffles
			shCombine := combineImages(shRow, shCol)
			imgs = append(imgs, shCombine)
			wg.Done()

		}(img)

	}
	wg.Wait()
	fmt.Println("\ndrawing...")
	gifFile := filepath.Join(outDir, "result.gif")
	createGif(gifFile, true, imgs...)

	fmt.Println("")
	fmt.Println(outDir)

}
