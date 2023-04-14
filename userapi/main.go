package main

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"time"
)

const store = `users.json`

type (
	User struct {
		CreatedAt   time.Time `json:"created_at"`
		DisplayName string    `json:"display_name"`
		Email       string    `json:"email"`
	}
	UserList  map[string]User
	UserStore struct {
		Increment int      `json:"increment"`
		List      UserList `json:"list"`
	}
)

var (
	UserNotFound = errors.New("user_not_found")
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(time.Now().String()))
	})

	r.Route("/api", func(r chi.Router) { // todo fix it
		r.Route("/v1", func(r chi.Router) {
			r.Route("/users", func(r chi.Router) {
				r.Get("/", searchUsers)
				r.Post("/", createUser)

				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", getUser)
					r.Patch("/", updateUser)
					r.Delete("/", deleteUser)
				})
			})
		})
	})

	http.ListenAndServe(":3334", r)
}

// func NewClient()
// func AddMiddleWare()
// func InitHandlers(r router) add groups
//

func searchUsers(w http.ResponseWriter, r *http.Request) {
	s := getUserStore()
	render.JSON(w, r, s.List)
}

type CreateUserRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func (c *CreateUserRequest) Bind(r *http.Request) error { return nil }

func createUser(w http.ResponseWriter, r *http.Request) { // TODO ALMOST FIXED
	s := getUserStore()

	request := CreateUserRequest{}
	if err := render.Bind(r, &request); err != nil {
		_ = render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	s.Increment++
	u := User{
		CreatedAt:   time.Now(),
		DisplayName: request.DisplayName,
		Email:       request.Email,
	}
	id := strconv.Itoa(s.Increment)
	s.List[id] = u

	err := toStore(&s)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"user_id": id,
	})
}

func getUser(w http.ResponseWriter, r *http.Request) {
	s := getUserStore()
	id := chi.URLParam(r, "id")
	render.JSON(w, r, s.List[id])
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
}

func (c *UpdateUserRequest) Bind(r *http.Request) error { return nil }

func updateUser(w http.ResponseWriter, r *http.Request) { // todo ALMOST FIXED
	s := getUserStore()

	request := UpdateUserRequest{}
	if err := render.Bind(r, &request); err != nil {
		_ = render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	id := chi.URLParam(r, "id")
	if !existUser(&s, id, w, r) {
		render.Status(r, http.StatusNotFound)
		return
	}

	u := s.List[id]
	u.DisplayName = request.DisplayName
	s.List[id] = u

	err := toStore(&s)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		return
	}
	render.Status(r, http.StatusNoContent)
}

func deleteUser(w http.ResponseWriter, r *http.Request) { // todo ALMOST FIXED
	s := getUserStore()
	id := chi.URLParam(r, "id")

	if !existUser(&s, id, w, r) {
		render.Status(r, http.StatusNotFound)
		return
	}
	delete(s.List, id)

	err := toStore(&s)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		return
	}
	render.Status(r, http.StatusNoContent)
}

type ErrResponse struct {
	Err            error `json:"-"`
	HTTPStatusCode int   `json:"-"`

	StatusText string `json:"status"`
	AppCode    int64  `json:"code,omitempty"`
	ErrorText  string `json:"error,omitempty"`
}

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

//func validRequest(request *render.Binder, w http.ResponseWriter, r *http.Request) error {
//	if err := render.Bind(r, *request); err != nil {
//		_ = render.Render(w, r, ErrInvalidRequest(err))
//		return err
//	}
//	return nil
//}

func existUser(userStore *UserStore, id string, w http.ResponseWriter, r *http.Request) bool {
	if _, ok := userStore.List[id]; !ok {
		_ = render.Render(w, r, ErrInvalidRequest(UserNotFound))
		return false
	}
	return true
}

func toStore(s *UserStore) error {
	b, _ := json.Marshal(*s)
	err := os.WriteFile(store, b, fs.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func getUserStore() UserStore {
	f, _ := os.ReadFile(store)
	s := UserStore{}
	_ = json.Unmarshal(f, &s)
	return s
}
