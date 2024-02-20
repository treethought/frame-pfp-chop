package farcaster

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"

	"github.com/treethought/impression-frame/util"
)

var NEYNAR_API_KEY = os.Getenv("API_KEY")

const API_URL = "https://hub-api.neynar.com"

type User struct {
	Username     string   `json:"username"`
	DisplayName  string   `json:"display_name"`
	PfpUrl       string   `json:"pfp_url"`
	Verfications []string `json:"verifications"`
	Profile      struct {
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

func GetUserName(name string, viewer uint64) (*User, error) {
	url := fmt.Sprintf("https://api.neynar.com/v2/farcaster/user/search?q=%s&viewer_fid=%d", name, viewer)
	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("api_key", NEYNAR_API_KEY)

	res, _ := http.DefaultClient.Do(req)

	var resp SearchResult
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	if len(resp.Result.Users) == 0 {
		return nil, fmt.Errorf("no users found")
	}
	return &resp.Result.Users[0], nil
}

func GetUser(fid uint64) (*User, error) {

	url := fmt.Sprintf("https://api.neynar.com/v2/farcaster/user/bulk?fids=%d", fid)
	fmt.Println(url)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("api_key", NEYNAR_API_KEY)

	res, _ := http.DefaultClient.Do(req)

	// d, _ := io.ReadAll(res.Body)
	// fmt.Println(string(d))

	var resp Users
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	if len(resp.Users) == 0 {
		return nil, fmt.Errorf("no users found")
	}
	fmt.Println(resp.Users[0].PfpUrl)
	return &resp.Users[0], nil

}

func GetOrLoadPFP(fid uint64) (image.Image, error) {
	known := Cache.GetPfpUrl(fid)
	if known != "" {
		cached, _, err := util.LoadImage(known)
		if err == nil {
			return cached, nil
		}
		img, _, err := util.FetchImage(known)
		if err != nil {
			log.Println("failed to fetch image: ", err)
			return nil, err
		}
		cachePath := fmt.Sprintf("%s/%s.png", cacheDir, util.EscapeURL(known))
		util.WriteImage(cachePath, img)
		return img, nil
	}

	log.Println("fetching pfp for fid: ", fid)
	user, err := GetUser(fid)
	if err != nil {
		log.Println("failed to get pfp: ", err)
		return nil, err
	}
	pfpUrl := user.PfpUrl
	log.Println("pfp url: ", pfpUrl)
	Cache.SetPfpUrl(fid, pfpUrl)

	cachePath := fmt.Sprintf("%s/%s.png", cacheDir, util.EscapeURL(pfpUrl))
	img, _, err := util.LoadImage(cachePath)
	if err != nil {

		img, _, err = util.FetchImage(pfpUrl)
		if err != nil {
			log.Println("failed to fetch image: ", err)
			return nil, err
		}
		cachePath := fmt.Sprintf("%s/%s.png", cacheDir, util.EscapeURL(pfpUrl))
		util.WriteImage(cachePath, img)
	}
	return img, nil
}
