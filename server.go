package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

const (
	BASE_URL = "https://api.nasa.gov/planetary/apod?api_key="
	COUNT_PARAM = "count=1"
	API_KEY_ENV_VAR = "NASA_API_KEY"
)

type rating int
type userEmail string
type imageURL string

type ImageStore struct {
	sync.Mutex
	url string
	imageStore map[imageURL]Image
}

type Image struct {
	Date string `json:"date"`
	Explanation string `json:"explanation"`
	Title string `json:"title"`
	Url string `json:"url"`
}

type Images []struct {
	Date string `json:"date"`
	Explanation string `json:"explanation"`
	Title string `json:"title"`
	Url string `json:"url"`
}

type User struct {
	sync.Mutex
	ratingsStore map[imageURL]rating
}

type userHandlers struct {
	sync.Mutex
	userStore map[userEmail]User
}

func newImageStore() *ImageStore {
	apiKey := os.Getenv(API_KEY_ENV_VAR)
	if apiKey == "" {
		panic("required environment variable NASA_API_KEY not set")
	} else {
		url := BASE_URL + apiKey + "&" + COUNT_PARAM
		return &ImageStore{
			url: url,
			imageStore: map[imageURL]Image{},
		}
	}
}

func (iStore ImageStore) imageHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(iStore.url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetching NASA image: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	
	var images Images	
	// API returns a JSON array, even though we're only querying for 1 image
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		panic(err)
	}
	image := images[0]

	// store image in "db"
	iStore.Lock()
	defer iStore.Unlock()
	url := imageURL(image.Url)
	iStore.imageStore[url] = image
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(image)
}

func main() {
	
	imgStore := newImageStore()

	http.HandleFunc("/image", imgStore.imageHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}