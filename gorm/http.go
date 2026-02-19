package fwork_server_gorm

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	fwork_server_orm "github.com/joabssilveira/GoQLite/core"
	"gorm.io/gorm"
)

// GET

func GormListHandler[T any](db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := GormGetListHttp[T](db, r, fwork_server_orm.Filter{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// POST

type CreatePayloadResolver[T any] func(r *http.Request) (T, error)

func BodyPayloadResolver[T any](r *http.Request) (T, error) {
	var payload T
	err := json.NewDecoder(r.Body).Decode(&payload)
	return payload, err
}

func GormCreateHandler[T any](
	db *gorm.DB,
	resolver ...CreatePayloadResolver[T],
) http.HandlerFunc {

	// default
	resolve := BodyPayloadResolver[T]
	if len(resolver) > 0 && resolver[0] != nil {
		resolve = resolver[0]
	}

	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := resolve(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// if err := db.Create(&payload).Error; err != nil {
		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		created, err := GormCreate(payload, db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}
}

// PUT

type UpdateStructResolver[T any] func(r *http.Request) (T, error)

func BodyStructResolver[T any](r *http.Request) (T, error) {
	var payload T
	err := json.NewDecoder(r.Body).Decode(&payload)
	return payload, err
}

func GormUpdateHandler[T any](
	db *gorm.DB,
	keyName string,
	resolver ...UpdateStructResolver[T],
) http.HandlerFunc {

	resolve := BodyStructResolver[T]
	if len(resolver) > 0 && resolver[0] != nil {
		resolve = resolver[0]
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		payload, err := resolve(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := GormUpdate(payload, id, db, keyName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(updated)
	}
}
