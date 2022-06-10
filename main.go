package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

var albums = []*Album{
	{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
	{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
	{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Route("/albums", func(r chi.Router) {
		r.Get("/", GetAlbums)
		r.Post("/", CreateAlbum)

		r.Route("/{albumID}", func(r chi.Router) {
			r.Use(AlbumCtx)
			r.Get("/", GetAlbum)
			r.Post("/", UpdateAlbum)
			r.Delete("/", DeleteAlbum)
		})
	})

	http.ListenAndServe(":3000", r)
}

func AlbumCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var album *Album
		var err error

		if albumID := chi.URLParam(r, "albumID"); albumID != "" {
			album, err = dbGetAlbum(albumID)
		} else {
			render.JSON(w, r, map[string]interface{}{"code": 404, "success": false, "err": err.Error()})
			return
		}
		if err != nil {
			render.JSON(w, r, map[string]interface{}{"code": 404, "success": false, "err": err.Error()})
			return
		}

		ctx := context.WithValue(r.Context(), "album", album)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetAlbums(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, albums)
}

func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var err error
	album := &Album{}
	render.Decode(r, album)

	_, err = dbCreateAlbum(album)

	if err != nil {
		render.JSON(w, r, map[string]interface{}{"code": 400, "success": false, "err": err.Error()})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, album)
}

func GetAlbum(w http.ResponseWriter, r *http.Request) {
	album := r.Context().Value("album").(*Album)
	render.JSON(w, r, album)
}

func UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	var err error
	oldAlbum := r.Context().Value("album").(*Album)
	album := &Album{}
	render.Decode(r, album)
	album, err = dbUpdateAlbum(oldAlbum.ID, album)
	if err != nil {
		render.JSON(w, r, map[string]interface{}{"code": 404, "success": false, "err": err.Error()})
		return
	}
	render.JSON(w, r, album)
}

func DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	var err error
	album := r.Context().Value("album").(*Album)
	album, err = dbDeleteAlbum(album.ID)
	if err != nil {
		render.JSON(w, r, map[string]interface{}{"code": 400, "success": false, "err": err.Error()})
		return
	}

	render.JSON(w, r, album)
}

//DATABASE
func dbGetAlbum(id string) (*Album, error) {
	for _, a := range albums {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("album not found.")
}

func dbCreateAlbum(album *Album) (string, error) {
	album.ID = fmt.Sprintf("%d", rand.Intn(100)+10)
	albums = append(albums, album)
	return album.ID, nil
}

func dbUpdateAlbum(id string, album *Album) (*Album, error) {
	for i, a := range albums {
		if a.ID == id {
			album.ID = id
			albums[i] = album
			return album, nil
		}
	}
	return nil, errors.New("alnum not found.")
}

func dbDeleteAlbum(id string) (*Album, error) {
	for i, a := range albums {
		if a.ID == id {
			albums = append((albums)[:i], (albums)[i+1:]...)
			return a, nil
		}
	}
	return nil, errors.New("alnum not found.")
}
