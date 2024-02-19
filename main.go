package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
)

const API_URL = "https://hub-api.neynar.com"
// var fid = 23
var fid uint64 = 0

// const fid = 04893

var API_KEY = os.Getenv("API_KEY")

// action enum using iota
// with values post, post_redirect, get
// and get_redirect

type Action int

const (
	ActionPOST Action = iota
	ActionPOSTRedirect
	ActionMint
	ActionLink
)

func (a Action) String() string {
	switch a {
	case ActionPOST:
		return "post"
	case ActionPOSTRedirect:
		return "post_redirect"
	case ActionMint:
		return "mint"
	case ActionLink:
		return "link"
	}
	return "unknown"
}

type UntrustedData struct {
	FID         uint64 `json:"fid"`
	URL         string `json:"url"`
	MessageHash string `json:"messageHash"`
	Timestamp   uint64 `json:"timestamp,uint64"`
	Network     int    `json:"network"`
	ButtonIndex int    `json:"buttonIndex"`
	InputText   string `json:"inputText"`
	CastId      struct {
		FID  uint64 `json:"fid"`
		Hash string `json:"hash"`
	} `json:"castId"`
}

type SignaturePacket struct {
	UntrustedData UntrustedData `json:"untrustedData"`
	TrustedData   struct {
		MessageBytes string `json:"messageBytes"`
	}
}

type Button struct {
	Label  []byte
	Action Action
	Target []byte
}

type Frame struct {
	frameV         string
	Image          string
	PostURL        string
	Buttons        []Button
	InputTextLabel string
}

func (f *Frame) Render(w io.Writer) {

	btns := ""
	for idx, b := range f.Buttons {
		i := idx + 1
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d" content="%s">\n`, i, b.Label)
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d:action" content="%s">\n`, i, b.Action.String())
		btns += fmt.Sprintf(`<meta property="fc:frame:button:%d:target" content="%s">\n`, i, b.Target)
	}

	inputTx := ""
	if f.InputTextLabel != "" {
		inputTx = fmt.Sprintf(`<meta property="fc:frame:input:text" content="%s">\n`, f.InputTextLabel)
	}

	// TODO aspect ratio

	resp := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>Frame</title>
  <meta property="og:image" content="%s">
  <meta property="fc:frame" content="%s">
  <meta property="fc:frame:image" content="%s">
  <meta property="fc:frame:post_url" content="%s">
  %s
  %s
</head>
  <body>
    HOWDY
  </body>
</html>`, f.Image, f.frameV, f.Image, f.PostURL, btns, inputTx)
	if _, err := w.Write([]byte(resp)); err != nil {
		panic(err)
	}

	// fmt.Println(resp)
}

func main() {

	f := Frame{
		frameV:  "vNext",
		Image:   "http://localhost:8080/images/cover.png",
		PostURL: "https://frame.seaborne.cloud/generate",
		Buttons: []Button{
			Button{
				Label:  []byte("Generate"),
				Action: ActionPOST,
				// Target: []byte("https://frame.seaborne.cloud/generate"),
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
		f.Render(w)
	})

	mux.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving image: ", r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	mux.HandleFunc("/results/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving result image: ", r.URL.Path[1:])
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	mux.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		log.Println("generating image")

		var packet SignaturePacket
		if err := json.NewDecoder(r.Body).Decode(&packet); err != nil {
			log.Println("failed to decode packet: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if fid == 0 {
			fid = packet.UntrustedData.FID
		}

		log.Println("fetching pfp for fid: ", fid)
		PfpUrl, err := getUserPFP(fid)
		if err != nil {
			log.Println("failed to get pfp: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println(PfpUrl)

		// get image from pfp url
		img, _, err := fetchImage(PfpUrl)
		if err != nil {
			log.Println("failed to fetch image: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// shuffle images
		log.Println("shuffling image")
		outDir := fmt.Sprintf("results/%d", fid)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			log.Println("failed to create output dir: ", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		result := runTransform(img, outDir)

		id := uuid.New().String()
		out := fmt.Sprintf("%s/%s.png", outDir, id)
		writeImage(out, result)
		log.Println("wrote image to: ", out)

		imgUrl := fmt.Sprintf("https://frame.seaborne.cloud/%s", out)

		frame := Frame{
			frameV:  "vNext",
			Image:   imgUrl,
			PostURL: imgUrl,
			Buttons: []Button{
				Button{
					Label:  []byte("View"),
					Action: ActionLink,
					Target: []byte(imgUrl),
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

func shuffleCombinePfps(imgs ...image.Image) image.Image {
	result := combineImages(imgs[0], imgs[1])
	return result
}

// returns rectangle centered in the pixture with x and y offset
func getRegion(img image.Image, x, y int) (x1, x2, y1, y2 int) {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	xoffset := width - (2 * x)
	yoffset := height - (2 * y)
	xq := xoffset / 2
	yq := yoffset / 2
	return xq, xq + x, yq, yq + y

}

func writeWithin(base image.Image, img image.Image) image.Image {

	// Get dimensions of the image
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Define the region in the middle of the image
	startX := width / 4
	endX := width * 3 / 4
	startY := height / 4
	endY := height * 3 / 4

	scaled := scaleImage(img, 10)
	result := writeImageWithinRegion(base, scaled, startX, endX, startY, endY)
	return result

}

func runTransform(img image.Image, outdir string) image.Image {
	shuffled := shufflePFPGrid(img, outdir)
	result := writeWithin(img, shuffled)
	return result
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

	// cid := runner.uploadResults()
	// // fmt.Println(cid)
	//
	// runner.generateMeta(cid)
	// runner.uploadMeta()

	// ids := runner.writeReuslts2()
	// cid := runner.uploadResults()
	//
	// runner.generateMeta(cid)
	// runner.uploadMeta()
	//
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
