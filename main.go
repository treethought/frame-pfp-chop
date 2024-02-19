package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/google/uuid"
	fc "github.com/treethought/impression-frame/farcaster"
	"github.com/treethought/impression-frame/gen"
	"github.com/treethought/impression-frame/util"
)

var (
	fid uint64 = uint64(rand.Intn(5000) + 1)
)

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/images/", serveImage)
	mux.HandleFunc("/results/", serveImage)
	mux.HandleFunc("/start", handleStart)
	mux.HandleFunc("/generate", handleGenerate)

	log.Println("starting server on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}

}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	start := fc.Frame{
		FrameV:  "vNext",
		Image:   "http://localhost:8080/images/cover.png",
		PostURL: "https://frame.seaborne.cloud/start",
		Buttons: []fc.Button{
			{
				Label:  []byte("Start"),
				Action: fc.ActionPOST,
			},
		},
	}
	start.Render(w)
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	log.Println("serving image: ", r.URL.Path[1:])
	http.ServeFile(w, r, r.URL.Path[1:])
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	log.Println("start request received")
	pfpUrl, err := fc.GetUserPFP(fid)
	if err != nil {
		log.Println("failed to get pfp: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get image from pfp url to cache
	_, err = fc.GetOrLoadPFP(fid)

	frame := fc.Frame{
		FrameV:  "vNext",
		Image:   pfpUrl,
		PostURL: "https://frame.seaborne.cloud/generate",
		Buttons: []fc.Button{
			{
				Label:  []byte("Slice and dice"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Shuffle"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Recombine"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Fractal"),
				Action: fc.ActionPOST,
			},
		},
	}

	frame.Render(w)
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {

	var packet fc.SignaturePacket
	if err := json.NewDecoder(r.Body).Decode(&packet); err != nil {
		log.Println("failed to decode packet: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if fid == 0 {
		fid = packet.UntrustedData.FID
	}

	var img image.Image
	var err error

	if r.Method == "POST" && r.URL.Query().Get("session") != "" {
		id := r.URL.Query().Get("session")
		path := fmt.Sprintf("results/%d/%s.png", fid, id)
		// read image from file
		log.Println("continue session")
		img, _, err = util.LoadImage(path)
		if err != nil {
			log.Println("failed to read image: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		img, err = fc.GetOrLoadPFP(fid)
	}

	outDir := fmt.Sprintf("results/%d", fid)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Println("failed to create output dir: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	var result image.Image
	switch packet.UntrustedData.ButtonIndex {
	case 1:
		result = runSliceAndDice(img, outDir)
	case 2:
		result = runShuffle(img, outDir)
	case 3:
		og := img
		if url := fc.Cache.GetPfpUrl(fid); url != "" {
			cached, err := fc.GetOrLoadPFP(fid)
			if err == nil {
				og = cached
			}
		}

		result = runRecombine(img, og, outDir)
	default:
		result = runTransform(img, outDir)
	}

	id := uuid.New().String()
	out := fmt.Sprintf("%s/%s.png", outDir, id)
	util.WriteImage(out, result)
	log.Println("wrote image to: ", out)

	imgUrl := fmt.Sprintf("https://frame.seaborne.cloud/%s", out)

	postURL := fmt.Sprintf("https://frame.seaborne.cloud/generate?session=%s", id)

	frame := fc.Frame{
		FrameV:  "vNext",
		Image:   imgUrl,
		PostURL: postURL,
		Buttons: []fc.Button{
			{
				Label:  []byte("Slice"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Shuffle"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Recombine"),
				Action: fc.ActionPOST,
			},
			{
				Label:  []byte("Fractal"),
				Action: fc.ActionPOST,
			},
		},
	}

	frame.Render(w)

}

func runSliceAndDice(img image.Image, outdir string) image.Image {
	log.Println("running slice and dice")
	sr, _ := gen.ShuffleImageRows(img)
	sc, _ := gen.ShuffleImageColumns(img)
	result := gen.CombineImages(sc, sr)
	return result
}

func runShuffle(img image.Image, outdir string) image.Image {
	log.Println("running shuffle")
	for i := 0; i < 2; i++ {
		sr, _ := gen.ShuffleImageRows(img)
		sc, _ := gen.ShuffleImageColumns(img)
		img = gen.CombineImages(sr, sc)
	}
	return img
}
func runRecombine(img image.Image, og image.Image, outdir string) image.Image {
	log.Println("running recombine")
	result := gen.WriteWithin(og, img, 80)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(img, og, 60)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(og, img, 40)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(img, og, 20)
	return result
}

func runRecombine2(img image.Image, outdir string) image.Image {
	log.Println("running recombine")
	result := gen.WriteWithin(img, img, 80)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(img, img, 60)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(img, img, 40)
	img = gen.CombineImages(img, result)
	result = gen.WriteWithin(img, img, 20)
	return result
}

func runTransform(img image.Image, outdir string) image.Image {
	log.Println("runTransform")
	// return img
	sr, _ := gen.ShuffleImageRows(img)
	sc, _ := gen.ShuffleImageColumns(img)
	result := gen.WriteWithin(sc, sr, 80)
	result = gen.CombineImages(result, sr)
	result = gen.WriteWithin(result, sc, 60)
	result = gen.WriteWithin(result, sr, 40)
	result = gen.WriteWithin(result, sc, 30)
	return result

	// bb := gen.CombineImages(sr, sc)
	// return gen.WriteWithin(bb, bb, 20)
}
