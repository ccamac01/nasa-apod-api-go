package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

const (
	BASE_URL         = "https://api.nasa.gov/planetary/apod?api_key="
	COUNT_PARAM      = "count=1"
	API_KEY_ENV_VAR  = "NASA_API_KEY"
	GET              = "GET"
	POST             = "POST"
	PUT              = "PUT"
	DELETE           = "DELETE"
	CONTENT_TYPE     = "content-type"
	APPLICATION_JSON = "application/json"
)

type rating int
type userEmail string
type imageURL string

type imageStore struct {
	sync.Mutex
	url   string
	store map[imageURL]Image
}

type Image struct {
	Date        string `json:"date"`
	Explanation string `json:"explanation"`
	Title       string `json:"title"`
	Url         string `json:"url"`
}

type Images []struct {
	Date        string `json:"date"`
	Explanation string `json:"explanation"`
	Title       string `json:"title"`
	Url         string `json:"url"`
}

func newImageStore() *imageStore {
	apiKey := os.Getenv(API_KEY_ENV_VAR)
	if apiKey == "" {
		panic("required environment variable NASA_API_KEY not set")
	} else {
		url := BASE_URL + apiKey + "&" + COUNT_PARAM
		return &imageStore{
			url:   url,
			store: map[imageURL]Image{},
		}
	}
}

func (i *imageStore) imageHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(i.url)
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
	i.Lock()
	defer i.Unlock()
	url := imageURL(image.Url)
	i.store[url] = image

	w.Header().Set(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(image)
}

type user struct {
	sync.Mutex
	store map[imageURL]rating
}

type users struct {
	sync.Mutex
	store map[userEmail]user
}

type User struct {
	Email string `json:"email"`
}

func newUser() user {
	return user{
		store: map[imageURL]rating{},
	}
}

func newUsers() *users {
	return &users{
		store: map[userEmail]user{},
	}
}

func (u *users) userHandlers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case POST:
		u.createUser(w, r)
		return
	case DELETE:
		u.deleteUser(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("METHOD NOT ALLOWED"))
		return
	}
}

func (u *users) createUser(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get(CONTENT_TYPE); ct != APPLICATION_JSON {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s' instead", ct)))
		return
	}

	var usr User
	if err := json.NewDecoder(r.Body).Decode(&usr); err != nil {
		panic(err)
	}

	usrEmail := userEmail(usr.Email)
	if usrEmail == "" || len(usrEmail) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'email' populated with a valid email as JSON in body request")))
		return
	}

	u.Lock()
	defer u.Unlock()
	u.store[usrEmail] = newUser()

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("user with email %v, successfully created", usrEmail)))
}

func (u *users) deleteUser(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get(CONTENT_TYPE); ct != APPLICATION_JSON {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s' instead", ct)))
		return
	}

	var usr User
	if err := json.NewDecoder(r.Body).Decode(&usr); err != nil {
		panic(err)
	}

	usrEmail := userEmail(usr.Email)
	if usrEmail == "" || len(usrEmail) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'email' populated with a valid email as JSON in body request")))
		return
	}

	u.Lock()
	defer u.Unlock()

	if _, ok := u.store[usrEmail]; ok {
		delete(u.store, usrEmail)
	}

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte(fmt.Sprintf("user with email %v, successfully deleted", usrEmail)))
}

func main() {

	i := newImageStore()
	u := newUsers()

	http.HandleFunc("/image", i.imageHandler)
	http.HandleFunc("/user", u.userHandlers)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
