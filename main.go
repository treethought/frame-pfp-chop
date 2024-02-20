package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/treethought/impression-frame/contract"
	fc "github.com/treethought/impression-frame/farcaster"
	"github.com/treethought/impression-frame/gen"
	"github.com/treethought/impression-frame/util"
)

var (
	BASE_URL = os.Getenv("BASE_URL")
)

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/images/", serveImage)
	mux.HandleFunc("/results/", serveImage)
	mux.HandleFunc("/start", handleStart)
	mux.HandleFunc("/generate", handleGenerate)

	log.Println("starting server on port 8080")
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		panic(err)
	}

}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	start := fc.Frame{
		FrameV:  "vNext",
		Image:   fmt.Sprintf("%s/images/cover.png", BASE_URL),
		PostURL: fmt.Sprintf("%s/start", BASE_URL),
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
	packet, err := getSignaturePacket(r)
	if err != nil {
		log.Println("failed to get signature packet: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fid := packet.UntrustedData.FID
	user, err := fc.GetUser(fid)
	if err != nil {
		log.Println("failed to get pfp: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	pfpUrl := user.PfpUrl

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

func getSignaturePacket(r *http.Request) (fc.SignaturePacket, error) {
	var packet fc.SignaturePacket
	if err := json.NewDecoder(r.Body).Decode(&packet); err != nil {
		log.Println("failed to decode packet: ", err)
		return packet, err
	}
	return packet, nil
}

func getSessionImg(r *http.Request, packet fc.SignaturePacket, fid uint64) (image.Image, error) {

	var img image.Image
	var err error

	if r.Method == "POST" && r.URL.Query().Get("session") != "" {
		id := r.URL.Query().Get("session")
		path := fmt.Sprintf("results/%d/%s.png", fid, id)
		log.Println("continue session")
		img, _, err = util.LoadImage(path)
		if err != nil {
			log.Println("failed to read image: ", err)
			return nil, err
		}
	} else {
		img, err = fc.GetOrLoadPFP(fid)
	}
	return img, err

}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	packet, err := getSignaturePacket(r)
	if err != nil {
		log.Println("failed to get signature packet: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	img, err := getSessionImg(r, packet, packet.UntrustedData.FID)
	if err != nil {
		log.Println("failed to get session image: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	outDir := fmt.Sprintf("results/%d", packet.UntrustedData.FID)
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
		if url := fc.Cache.GetPfpUrl(packet.UntrustedData.FID); url != "" {
			cached, err := fc.GetOrLoadPFP(packet.UntrustedData.FID)
			if err == nil {
				og = cached
			}
		}
		result = runRecombine(img, og, outDir)
	case 4:
		frame, err := handleMint(r.Context(), packet, img)
		if err != nil {
			log.Println("failed to mint: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		frame.Render(w)
		return
	default:
		result = runTransform(img, outDir)
	}

	id := uuid.New().String()
	out := fmt.Sprintf("%s/%s.png", outDir, id)
	util.WriteImage(out, result)
	log.Println("wrote image to: ", out)

	imgUrl := fmt.Sprintf("%s/%s", BASE_URL, out)

	postURL := fmt.Sprintf("%s/generate?session=%s", BASE_URL, id)

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
				Label:  []byte("Mint"),
				Action: fc.ActionPOST,
				Target: []byte(fmt.Sprintf("%s/mint", BASE_URL)),
			},
		},
	}

	frame.Render(w)

}

func handleMint(ctx context.Context, packet fc.SignaturePacket, img image.Image) (*fc.Frame, error) {
	user, err := fc.GetUser(packet.UntrustedData.FID)
	if err != nil {
		log.Println("failed to get user: ", err)
		return nil, err
	}
	c, err := contract.NewContract()
	if err != nil {
		log.Println("failed to create contract: ", err)
		return nil, err
	}

	tx, err := c.Mint(ctx, img, user)
	if err != nil {
		log.Println("failed to mint: ", err)
		return nil, err
	}
	result, _ := json.Marshal(&tx)
	fmt.Println(string(result))

	explorerUrl := fmt.Sprintf("https://sepolia.explorer.zora.energy/tx/%s", tx.Hash().Hex())

	frame := &fc.Frame{
		FrameV:  "vNext",
		Image:   fmt.Sprintf("%s/images/cover.png", BASE_URL),
		PostURL: fmt.Sprintf("%s/start", BASE_URL),
		Buttons: []fc.Button{
			{
				Label:  []byte("View"),
				Action: fc.ActionLink,
				Target: []byte(explorerUrl),
			},
		},
	}

	return frame, nil

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
