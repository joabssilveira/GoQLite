package goqlitegorm

import (
	"encoding/json"
	"net/http"

	"github.com/joabssilveira/GoQLite/goqlite"
	"gorm.io/gorm"
)

// GET

func GetHandler[T any](db *gorm.DB, dbUtils goqlite.DbUtils) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := ReadFromHttpRequestParams[T](db, r, goqlite.Filter{}, dbUtils)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
