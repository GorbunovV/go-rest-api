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

type AlbumResponse struct {
	*Album
}

type AlbumRequest struct {
	*Album
	ProtectedID string `json:"id"`
}

type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}

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
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "album", album)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetAlbums(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewAlbumsResponse(albums)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	data := &AlbumRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	album := data.Album
	dbCreateAlbum(album)

	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewAlbumResponse(album))
}

func GetAlbum(w http.ResponseWriter, r *http.Request) {
	album := r.Context().Value("album").(*Album)

	if err := render.Render(w, r, NewAlbumResponse(album)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

func UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	album := r.Context().Value("album").(*Album)

	data := &AlbumRequest{Album: album}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	album = data.Album
	dbUpdateAlbum(album.ID, album)

	render.Render(w, r, NewAlbumResponse(album))
}

func DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	var err error
	album := r.Context().Value("album").(*Album)

	album, err = dbDeleteAlbum(album.ID)

	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	render.Render(w, r, NewAlbumResponse(album))
}

func (rd *AlbumResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (a *AlbumRequest) Bind(r *http.Request) error {
	if a.Album == nil {
		return errors.New("missing required Album fields.")
	}
	return nil
}

func NewAlbumResponse(album *Album) *AlbumResponse {
	return &AlbumResponse{Album: album}
}

func NewAlbumsResponse(albums []*Album) []render.Renderer {
	list := []render.Renderer{}

	for _, album := range albums {
		list = append(list, NewAlbumResponse(album))
	}

	return list
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

//ERROR
func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}
