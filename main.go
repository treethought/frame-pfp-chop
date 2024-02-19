package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/google/uuid"
)

const API_URL = "https://hub-api.neynar.com"

// genertae rand uint in range 1-5000

var (
	API_KEY        = os.Getenv("API_KEY")
	fid     uint64 = uint64(rand.Intn(5000) + 1)

	cacheDir = "tmp/framecache"
)

func main() {
	os.MkdirAll(cacheDir, 0755)

	start := Frame{
		frameV:  "vNext",
		Image:   "http://localhost:8080/images/cover.png",
		PostURL: "https://frame.seaborne.cloud/start",
		Buttons: []Button{
			Button{
				Label:  []byte("Start"),
				Action: ActionPOST,
			},
		},
	}

	// start std api server with dummy index handler

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			log.Println("POST request received")
			d, _ := io.ReadAll(r.Body)
			fmt.Println(string(d))
		}
		log.Println("Request received")
		start.Render(w)
	})

	mux.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving image: ", r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	mux.HandleFunc("/results/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving result image: ", r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		log.Println("start request received")
		pfpUrl, err := getUserPFP(fid)
		if err != nil {
			log.Println("failed to get pfp: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// get image from pfp url
		img, _, err := fetchImage(pfpUrl)
		if err != nil {
			log.Println("failed to fetch image: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// escape the pfp url to use as a file name

		cachePath := fmt.Sprintf("%s/%s", cacheDir, escapeURL(pfpUrl))
		writeImage(cachePath, img)

		frame := Frame{
			frameV:  "vNext",
			Image:   pfpUrl,
			PostURL: "https://frame.seaborne.cloud/generate",
			Buttons: []Button{
				Button{
					Label:  []byte("Slice and dice"),
					Action: ActionPOST,
				},
				Button{
					Label:  []byte("Shuffle"),
					Action: ActionPOST,
				},
				Button{
					Label:  []byte("Recombine"),
					Action: ActionPOST,
				},
				Button{
					Label:  []byte("Broken Mirror"),
					Action: ActionPOST,
				},
			},
		}

		frame.Render(w)
	})

	mux.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {

		var packet SignaturePacket
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
			img, _, err = loadImage(path)
			if err != nil {
				log.Println("failed to read image: ", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			log.Println("fetching pfp for fid: ", fid)
			PfpUrl, err := getUserPFP(fid)
			if err != nil {
				log.Println("failed to get pfp: ", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Println("pfp url: ", PfpUrl)

			cachePath := fmt.Sprintf("%s/%s.png", cacheDir, escapeURL(PfpUrl))

			img, _, err = loadImage(cachePath)
			if err != nil {

				// get image from pfp url
				img, _, err = fetchImage(PfpUrl)
				if err != nil {
					log.Println("failed to fetch image: ", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				log.Println("fetched image")
			}
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
			result = runRecombine(img, outDir)
		default:
			result = runTransform(img, outDir)
		}

		id := uuid.New().String()
		out := fmt.Sprintf("%s/%s.png", outDir, id)
		writeImage(out, result)
		log.Println("wrote image to: ", out)

		imgUrl := fmt.Sprintf("https://frame.seaborne.cloud/%s", out)

		postURL := fmt.Sprintf("https://frame.seaborne.cloud/generate?session=%s", id)

		frame := Frame{
			frameV:  "vNext",
			Image:   imgUrl,
			PostURL: postURL,
			Buttons: []Button{
				Button{
					Label:  []byte("Slice and dice"),
					Action: ActionPOST,
					// Target: []byte("https://frame.seaborne.cloud/generate"),
				},
				Button{
					Label:  []byte("Shuffle"),
					Action: ActionPOST,
				},
				Button{
					Label:  []byte("Recombine"),
					Action: ActionPOST,
				},
				Button{
					Label:  []byte("Butcher it"),
					Action: ActionPOST,
				},
			},
		}

		frame.Render(w)

	})

	log.Println("starting server on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}

}

func runSliceAndDice(img image.Image, outdir string) image.Image {
	log.Println("running slice and dice")
	sr, _ := shuffleImageRows(img)
	sc, _ := shuffleImageColumns(img)
	result := combineImages(sc, sr)
	return result
}

func runShuffle(img image.Image, outdir string) image.Image {
	log.Println("running shuffle")
	for i := 0; i < 2; i++ {
		sr, _ := shuffleImageRows(img)
		sc, _ := shuffleImageColumns(img)
		img = combineImages(sr, sc)
	}
	return img
}

func runRecombine(img image.Image, outdir string) image.Image {
	log.Println("running recombine")
	result := writeWithin(img, img, 80)
	img = combineImages(img, result)
	result = writeWithin(img, img, 60)
	img = combineImages(img, result)
	result = writeWithin(img, img, 40)
	img = combineImages(img, result)
	result = writeWithin(img, img, 20)
	return result
}

func runTransform(img image.Image, outdir string) image.Image {
	log.Println("runTransform")
	// return img
	sr, _ := shuffleImageRows(img)
	sc, _ := shuffleImageColumns(img)
	result := writeWithin(sc, sr, 80)
	result = combineImages(result, sr)
	result = writeWithin(result, sc, 60)
	result = writeWithin(result, sr, 40)
	result = writeWithin(result, sc, 30)
	return result
	bb := combineImages(sr, sc)
	return writeWithin(bb, bb, 20)

	// shufflebase := shuffleCombinePfps(img, img)
	// result := writeWithin(img, shuffled, 50)
	// return result
}

func shufflePFPGrid(img image.Image, outdir string) image.Image {
	runner := newRunner(outdir)
	runner.transforms = make(map[string]transformer)
	runner.transforms["horizontal"] = shuffleImageColumns
	runner.transforms["vertical"] = shuffleImageRows
	runner.transforms["vertical2"] = shuffleImageRows
	runner.run(img)
	if len(runner.results) < 3 {
		return nil
	}
	result := combineImages(runner.results[0], runner.results[1])
	result = combineImages(result, runner.results[2])
	result = combineImages(result, runner.results[1])
	// runner.writeReuslts()

	return result

}

type User struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	PfpUrl      string `json:"pfp_url"`
	Profile     struct {
		Bio struct {
			Text string `json:"text"`
		} `json:"bio"`
	} `json:"profile"`
}

type Users struct {
	Users []User `json:"users"`
}

type SearchResult struct {
	Result struct {
		Users []User
	}
}

func getUserPFPByName(name string, viewer uint64) (string, error) {
	url := fmt.Sprintf("https://api.neynar.com/v2/farcaster/user/search?q=%s&viewer_fid=%d", name, viewer)
	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("api_key", API_KEY)

	res, _ := http.DefaultClient.Do(req)

	var resp SearchResult
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}
	if len(resp.Result.Users) == 0 {
		return "", fmt.Errorf("no users found")
	}
	return resp.Result.Users[0].PfpUrl, nil
}

func getUserPFP(fid uint64) (string, error) {

	url := fmt.Sprintf("https://api.neynar.com/v2/farcaster/user/bulk?fids=%d", fid)
	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("api_key", API_KEY)

	res, _ := http.DefaultClient.Do(req)

	// d, _ := io.ReadAll(res.Body)
	// fmt.Println(string(d))

	var resp Users
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}
	if len(resp.Users) == 0 {
		return "", fmt.Errorf("no users found")
	}
	fmt.Println(resp.Users[0].PfpUrl)
	return resp.Users[0].PfpUrl, nil

}
