package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/savaki/dynamodbstore"
)

func main() {
	store, err := dynamodbstore.New(dynamodbstore.Path("/"), dynamodbstore.HTTPOnly())
	if err != nil {
		log.Fatalln(err)
	}

	router := mux.NewRouter()
	router.Path("/").HandlerFunc(withSession(store, "blah", hello))

	log.Fatalln(http.ListenAndServe(":3001", router))
}

func withSession(store sessions.Store, name string, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, _ := store.Get(req, name)
		if session.IsNew {
			session.Save(req, w)
		}
		defer session.Save(req, w)

		fn(w, req)
	}
}

func hello(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "hello world")
}
