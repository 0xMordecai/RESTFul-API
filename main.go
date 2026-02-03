package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Role     string
	Username string
	Password string
}
type Session struct {
	Expires  time.Time
	Username string
}
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ShoppingList struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Items []string `json:"items"`
}
type ShoppingListPatch struct {
	Name  *string  `json:"name"`
	Items []string `json:"items"`
}
type ListPushAction struct {
	Item string `json:"item"`
}

var allData []ShoppingList
var sessions = map[string]*Session{}
var allUsers = map[string]*User{
	"admin": {"admin", "admin", "password"},
	"user":  {"user", "user", "password"},
}

func main() {
	// The login endpoint
	http.HandleFunc("POST /login", handleLogin)
	// The creation endpoint
	http.HandleFunc("POST /v1/lists", adminRequired(handleCreateList))
	// The list endpoint
	http.HandleFunc("GET /v1/lists", authRequired(handleListLists))
	// The delete endpoint
	http.HandleFunc("DELETE /v1/lists/{id}", adminRequired(handleDeleteList))
	//	The update endpoint
	http.HandleFunc("PUT /v1/lists/{id}", adminRequired(handleUpdateList))
	//	The Patch endpoint
	http.HandleFunc("PATCH /v1/lists/{id}", adminRequired(handlePatchList))
	//	The retriever endpoint
	http.HandleFunc("GET /v1/lists/{id}", authRequired(handleGetList))
	//	the add-to-list action endpoint
	http.HandleFunc("POST /v1/lists/{id}/push", adminRequired(handleListPush))
	fmt.Println("listening on port :8888")
	http.ListenAndServe(":8888", nil)
}

// Authentication Handlers
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var data LoginRequest
	json.NewDecoder(r.Body).Decode(&data)
	user := allUsers[data.Username]
	if user != nil && user.Password == data.Password {
		token := strconv.Itoa(rand.IntN(100000000000))
		sessions[token] = &Session{
			Expires:  time.Now().Add(7 * 24 * time.Hour),
			Username: user.Username,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
	w.WriteHeader(http.StatusUnauthorized)
}

// Authorization middleware -
/*
This middleware checks the Authorization header to get the user token, then checks if the token
is correctly formatted, if the token exists in the sessions and the session is valid, and, finally, if the
user exists. If everything is okay, it lets the request pass to the next handler. If not, it will return
a 401 Unauthorized status code
*/
func authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token = token[7:]
		if sessions[token] == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if sessions[token].Expires.Before(time.Now()) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		user := allUsers[sessions[token].Username]
		if user == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func adminRequired(next http.HandlerFunc) http.HandlerFunc {
	return authRequired(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]
		user := allUsers[sessions[token].Username]
		if user.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// CRUD Handlers
func handleCreateList(w http.ResponseWriter, r *http.Request) {
	var list ShoppingList
	//	Unmarshall request's body
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//	if everything went well ,we will store information in our allData var and return the newly created instance
	allData = append(allData, list)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(list)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleListLists(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(allData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleDeleteList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, list := range allData {
		if strconv.Itoa(list.ID) == id {
			allData = append(allData[:i], allData[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	http.Error(w, "List not found", http.StatusNotFound)
}

func handleUpdateList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, list := range allData {
		if strconv.Itoa(list.ID) == id {
			var updatedList ShoppingList
			err := json.NewDecoder(r.Body).Decode(&updatedList)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			allData[i] = updatedList
			if err := json.NewEncoder(w).Encode(updatedList); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(w, "List not found", http.StatusNotFound)
}

func handlePatchList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, list := range allData {
		if strconv.Itoa(list.ID) == id {
			var patch ShoppingListPatch
			err := json.NewDecoder(r.Body).Decode(&patch)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if patch.Name != nil {
				list.Name = *patch.Name
			}
			if patch.Items != nil {
				list.Items = patch.Items
			}
			allData[i] = list
			err = json.NewEncoder(w).Encode(list)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(w, "List not found", http.StatusNotFound)
}

func handleGetList(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, list := range allData {
		if strconv.Itoa(list.ID) == id {
			data, err := json.Marshal(list)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = w.Write(data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(w, "List not found", http.StatusNotFound)
}

func handleListPush(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, list := range allData {
		if strconv.Itoa(list.ID) == id {
			var item ListPushAction
			err := json.NewDecoder(r.Body).Decode(&item)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			list.Items = append(list.Items, item.Item)
			allData[i] = list
			err = json.NewEncoder(w).Encode(list)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(w, "List not found", http.StatusNotFound)
}
