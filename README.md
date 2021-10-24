# nasa-apod-api-go
> Light JSON API for storing user ratings of NASA's Astronomy Picture of the Day (APOD).

To run this server you must have access to a NASA API key. One can be generated here:
https://api.nasa.gov/
Store this API key as an environment variable `NASA_API_KEY` before starting the server.

## How-to
`git-clone` this repository\
`cd` into repo directory\
run `go run server.go` and the server will start up on `localhost:8080`\
For examples on how to send requests, import the full [Postman collection](https://www.getpostman.com/collections/ce61b6ca3b2bd2cca4dd) locally

## Functionality
* Fetch and save a NASA picture to the database
* Create and delete users (only an email field is needed)
* Save a 1-5 star rating of a picture for a user
* Update a picture rating for a user
* Delete a user rating
* Get all of a user's ratings

## Requirements

This REST API must match a few requirements:
* [x] `GET /image` returns an image (JSON) from NASA's APOD API and stores in the db
* [x] `POST /user` creates a new user, returns error if email not included in JSON body 
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com"
    }
    
    ```
* [x] `DELETE /user` deletes a user, returns error if email not included in JSON body 
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com"
    }
    
    ```
* [x] `POST /rating` saves the rating for the specified image and user, returns error if email, imageID & rating are not included in JSON body 
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com",
        "imageURL": "https://apod.nasa.gov/apod/image/some_image_number_here/some_image_name_here.jpg",
        "rating": 5
    }
    
    ```
* [x] `GET /rating` returns all ratings associated with the user email, returns error if email not included in request params
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com"
    }
    
    ```
* [x] `PUT /rating` updates the rating associated with the image and user, returns error if email, imageID & rating are not included in JSON body 
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com",
        "imageURL": "https://apod.nasa.gov/apod/image/some_image_number_here/some_image_name_here.jpg",
        "rating": 4
    }
    
    ```
* [x] `DELETE /rating` deletes the rating associated with the image and user, returns error if email & imageID are not included in JSON body 
    * Body request requirements: 
    ```json
    {
        "email": "YOUR_EMAIL@mail.com",
        "imageURL": "https://apod.nasa.gov/apod/image/some_image_number_here/some_image_name_here.jpg"
    }
    
    ```

### Data Types

These fields must be included as JSON in the body of POST/PUT/DELETE requests (and in the GET request - where required)\
`email`: string containing the email associated with a user\
`imageURL`: string containing the `url` associated with an image (see down below)\
`rating`: an integer ranging from 1 to 5 (inclusive)\

An image object should look like this:
```json
{
    "date": "2021-10-23",
    "explanation": "Put on your red/blue glasses and float next to asteroid 101955 Bennu. Shaped like a spinning toy top with boulders littering its rough surface, the tiny Solar System world is about one Empire State Building (less than 500 meters) across. Frames used to construct this 3D anaglyph were taken by PolyCam on the OSIRIS_REx spacecraft on December 3, 2018 from a distance of about 80 kilometers. With a sample from the asteroid's rocky surface on board, OSIRIS_REx departed Bennu's vicinity this May and is now enroute to planet Earth. The robotic spacecraft is scheduled to return the sample to Earth in September 2023.",
    "title": "3D Bennu",
    "url": "https://apod.nasa.gov/apod/image/2110/ana03BennuVantuyne1024c.jpg"
}
```

### Persistence

There is no persistence, a temporary in-mem story is being utilized.

### RESTful Architecture
Miro board: https://miro.com/app/board/o9J_loAMrdw=/?invite_link_id=796923605486
![alt text](https://github.com/ccamac01/nasa-apod-api-go/blob/main/nasa-apod-api-restful-architecture.png?raw=true)
