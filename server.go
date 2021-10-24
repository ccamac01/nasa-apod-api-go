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

type user struct {
	sync.Mutex
	store map[imageURL]rating
}

type users struct {
	sync.Mutex
	store map[userEmail]user
}


// for JSON marshal/unmarshal
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

type User struct {
	Email    string `json:"email"`
	ImageURL string `json:"imageURL"`
	Rating   int    `json:"rating"`
}

// newImageStore instantiates imageStore and returns a pointer to it
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

// newUser instantiates and returns a new user
func newUser() user {
	return user{
		store: map[imageURL]rating{},
	}
}

// newUsers instantiates users and returns a pointer to it
func newUsers() *users {
	return &users{
		store: map[userEmail]user{},
	}
}

// imageHandler is responsible for requests sent to the /image endpoint
// it fetches an image from NASA's APOD API, stores it locally, and returns it via response
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

// userHandlers is responsible for routing requests from the /user endpoint
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

// createUser creates a new user in the user store
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
	if _, ok := u.store[usrEmail]; ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("user with email %s already exists", usrEmail)))
		return
	} else {
		u.store[usrEmail] = newUser()
	}

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("user with email %v, successfully created", usrEmail)))
}

// deleteUser deletes a user from the user store
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

// ratingHandlers is responsible for routing the requests from the /rating endpoint
func (u *users) ratingHandlers(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get(CONTENT_TYPE); ct != APPLICATION_JSON {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s' instead", ct)))
		return
	}

	// switch statement checking the type of request
	switch r.Method {
	case GET:
		u.getRatings(w, r)
		return
	case PUT:
		u.updateRating(w, r)
		return
	case POST:
		u.saveRating(w, r)
		return
	case DELETE:
		u.deleteRating(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("METHOD NOT ALLOWED"))
		return
	}
}

// saveRating stores a rating associated with an image, for the specified user
func (u *users) saveRating(w http.ResponseWriter, r *http.Request) {
	// check for email in body response
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
	iURL := imageURL(usr.ImageURL)
	if iURL == "" || len(usrEmail) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'imageURL' populated with a valid image URL as JSON in body request")))
		return
	}
	iRating := rating(usr.Rating)
	if iRating < 1 || iRating > 5 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'rating' populated with a valid integer rating 1-5 as JSON in body request")))
		return
	}

	// read user from store list
	u.Lock()
	existingUser, ok := u.store[usrEmail]
	u.Unlock()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("user with email %s does not exist", usrEmail)))
		return
	}

	// check if image already exists with a rating
	existingUser.Lock()
	if _, ok := existingUser.store[iURL]; ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("image with url %s already exists - send PUT request to update rating", iURL)))
		return
	} else {
		existingUser.store[iURL] = iRating
	}
	existingUser.Unlock()

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("rating successfully saved")))
}

// getRatings returns all image ratings associated with a user
func (u *users) getRatings(w http.ResponseWriter, r *http.Request) {
	// check for email in body response
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

	// read user from store list
	u.Lock()
	existingUser, ok := u.store[usrEmail]
	u.Unlock()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("user with email %s does not exist", usrEmail)))
		return
	}

	existingUser.Lock()
	w.Header().Set(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existingUser.store)
	existingUser.Unlock()
}

// updateRating updates the rating of an image associated with a user
func (u *users) updateRating(w http.ResponseWriter, r *http.Request) {
	// check for email in body response
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
	iURL := imageURL(usr.ImageURL)
	if iURL == "" || len(usrEmail) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'imageURL' populated with a valid image URL as JSON in body request")))
		return
	}
	iRating := rating(usr.Rating)
	if iRating < 1 || iRating > 5 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'rating' populated with a valid integer rating 1-5 as JSON in body request")))
		return
	}

	// read user from store list
	u.Lock()
	existingUser, ok := u.store[usrEmail]
	u.Unlock()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("user with email %s does not exist", usrEmail)))
		return
	}

	// check if image already exists with a rating
	existingUser.Lock()
	if _, ok := existingUser.store[iURL]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("image with url %s doesn't exist - send POST request to save rating", iURL)))
		return
	} else {
		// update rating
		existingUser.store[iURL] = iRating
	}
	existingUser.Unlock()

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte(fmt.Sprintf("rating successfully updated")))
}

// deleteRating deletes a rating associated with an image for a specified user
func (u *users) deleteRating(w http.ResponseWriter, r *http.Request) {
	// check for email in body response
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
	iURL := imageURL(usr.ImageURL)
	if iURL == "" || len(usrEmail) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("need field 'imageURL' populated with a valid image URL as JSON in body request")))
		return
	}

	// read user from store list
	u.Lock()
	existingUser, ok := u.store[usrEmail]
	u.Unlock()
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("user with email %s does not exist", usrEmail)))
		return
	}

	// check if image already exists with a rating
	existingUser.Lock()
	if _, ok := existingUser.store[iURL]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("image with url %s doesn't exist", iURL)))
		return
	} else {
		// delete rating
		delete(existingUser.store, iURL)
	}
	existingUser.Unlock()

	w.Header().Add(CONTENT_TYPE, APPLICATION_JSON)
	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte(fmt.Sprintf("rating successfully deleted")))
}

func main() {

	i := newImageStore()
	u := newUsers()

	http.HandleFunc("/image", i.imageHandler)
	http.HandleFunc("/user", u.userHandlers)
	http.HandleFunc("/rating", u.ratingHandlers)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
