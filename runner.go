package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	api "github.com/ipfs/go-ipfs-api"
	"github.com/mccutchen/palettor"
)

type transformer func(img image.Image) (image.Image, error)

type Runner struct {
	outDir       string
	currentImage image.Image
	transforms   map[string]transformer
	palletes     []*palettor.Palette
	results      []image.Image
	attrMap      []map[string]interface{}
	resultCIDs   []string
	ipfs         *api.Shell
}

func newRunner(outDir string) *Runner {
	r := new(Runner)
	r.ipfs = api.NewLocalShell()
	r.outDir = outDir
	return r
}

func (r *Runner) run(img image.Image) {
	fmt.Print(".")
	r.currentImage = img
	for name, trans := range r.transforms {
		newImg, err := trans(img)
		if err != nil {
			log.Fatal(err)
		}

		sourceHash := hashImage(img)
		ambiguieHash := hashImage(newImg)
		palette := getPalette(newImg)

		attrs := map[string]interface{}{
			"orientation":    name,
			"source_hash":    sourceHash,
			"ambiguate_hash": ambiguieHash,
			"palette":        palette.Entries(),
		}
		r.results = append(r.results, newImg)
		r.attrMap = append(r.attrMap, attrs)
		r.currentImage = newImg
	}

	// shuffle results

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(r.results), func(i, j int) {
		r.results[i], r.results[j] = r.results[j], r.results[i]
		r.attrMap[i], r.attrMap[j] = r.attrMap[j], r.attrMap[i]
	})

}

func (r *Runner) mixPallete(img image.Image) {
	pidx := rand.Intn(len(r.palletes))
	p := r.palletes[pidx]
	newImg, err := applyPallate(img, p)
	if err != nil {
		log.Fatal(err)
	}
	r.results = append(r.results, newImg)

	// shuffle
	sc, err := shuffleImageColumns(newImg)
	if err != nil {
		log.Fatal(err)
	}
	sr, err := shuffleImageRows(newImg)
	if err != nil {
		log.Fatal(err)
	}
	fin := combineImages(sc, sr)
	r.results = append(r.results, fin)

}

func (r *Runner) writeReuslts2() []string {
	ids := []string{}
	fmt.Println("\nwriting results")
	for _, img := range r.results {
		id := uuid.New()
		out := filepath.Join(r.outDir, fmt.Sprintf("%s.png", id.String()))
		ids = append(ids, id.String())
		writeImage(out, img)
	}
	return ids
}

func (r *Runner) writeReuslts() {
	fmt.Println("\nwriting results")
	for _, img := range r.results {
		id := uuid.New()
		out := filepath.Join(r.outDir, fmt.Sprintf("%s.png", id.String()))
		writeImage(out, img)
	}
}

func (r *Runner) createGif() {
	fmt.Println("\ndrawing...")
	gifFile := filepath.Join(r.outDir, "result.gif")
	createGif(gifFile, false, r.results...)
}

func (r *Runner) uploadResults() string {
	fmt.Println("uploading results to ipfs")
	res, err := r.ipfs.AddDir(r.outDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("image dir CID", res)
	return res
}

type Attribute struct {
	TraitType string      `json:"trait_type,omitempty"`
	Value     interface{} `json:"value,omitempty"`
}

type Metadata struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	ExternalURL string      `json:"external_url,omitempty"`
	Image       string      `json:"image,omitempty"`
	Attributes  []Attribute `json:"attributes"`
}

func (r *Runner) generateMeta(dirCid string) {
	fmt.Println("generating metadata")
	metaDir := filepath.Join(r.outDir, "meta")
	err := os.MkdirAll(metaDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	meta := []Metadata{}

	for idx, _ := range r.results {
		i := idx + 1

		attributes := []Attribute{}
		for k, v := range r.attrMap[idx] {
			a := Attribute{
				TraitType: k,
				Value:     v,
			}
			attributes = append(attributes, a)
		}

		url := fmt.Sprintf("ipfs://%s/%d.png", dirCid, i)
		m := Metadata{
			Name:        fmt.Sprintf("Ambiguate #%d", i),
			Description: fmt.Sprintf("ambiguity of %d", i),
			ExternalURL: url,
			Image:       url,
			Attributes:  attributes,
		}
		meta = append(meta, m)

		data, err := json.Marshal(&m)
		if err != nil {
			log.Fatal(err)
		}

		metaFile := filepath.Join(metaDir, fmt.Sprint(i))
		err = ioutil.WriteFile(metaFile, data, 0755)
		if err != nil {
			log.Fatal(err)
		}

	}

	data, err := json.Marshal(&meta)
	if err != nil {
		log.Fatal(err)
	}
	assetsFile := filepath.Join(r.outDir, "assets.json")

	err = ioutil.WriteFile(assetsFile, data, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func (r *Runner) uploadMeta() string {
	metaDir := filepath.Join(r.outDir, "meta")
	fmt.Println("uploading metadata to ipfs: ", metaDir)
	res, err := r.ipfs.AddDir(metaDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("baseUri CID: ", res)
	return res
}
