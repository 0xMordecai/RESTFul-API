package main

import (
	"database/sql"
	"math/rand"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
	db *sql.DB
}

// Adding persistence to the API

func NewRepository(database string) (*Repository, error) {
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		return nil, err
	}
	return &Repository{db}, nil
}

// Set up the database tables
func (r *Repository) Init() error {
	if _, err := r.db.Exec("CREATE TABLE IF NOT EXISTS users (role VARCHAR, username VARCHAR PRIMARY KEY, password VARCHAR)"); err != nil {
		return err
	}
	if _, err := r.db.Exec("CREATE TABLE IF NOT EXISTS sessions (token VARCHAR PRIMARY KEY, expires TIMESTAMP, username VARCHAR)"); err != nil {
		return err
	}
	if _, err := r.db.Exec("CREATE TABLE IF NOT EXISTS shoping_lists(id VARCHAR PRIMARY KEY, name VARCHAR, items TEXT )"); err != nil {
		return err
	}
	return nil
}

// method to store session
func (r *Repository) AddSession(username string) (*Session, error) {
	token := strconv.Itoa(rand.Intn(100000000000))
	session := Session{Token: token, Expires: time.Now().Add(7 * 24 * time.Hour), Username: username}
	query := sq.Insert("sessions").Columns("token", "expires", "username").Values(session.Token, session.Expires, session.Username)
	_, err := query.RunWith(r.db).Exec()
	if err != nil {
		return nil, err
	}

	return &session, nil
}
func (r *Repository) GetSession(token string) (*Session, error) {
	query := sq.Select("token", "expires", "username").From("sessions").Where(sq.Eq{"token": token}, sq.Gt{"expires": time.Now()})
	row := query.RunWith(r.db).QueryRow()
	session := Session{}
	if err := row.Scan(&session.Token, &session.Expires, &session.Username); err != nil {
		return nil, err
	}
	return &session, nil
}

// method to patch the shopping list
func (r *Repository) PatchShoppingList(id string, patch *ShoppingListPatch) error {
	query := sq.Update("shoping_lists").Where(sq.Eq{"id": id})
	if patch.Name != nil {
		query = query.Set("name", *patch.Name)
	}
	if patch.Items != nil {
		query = query.Set("items", strings.Join(patch.Items, ","))
	}
	_, err := query.RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	return nil
}
