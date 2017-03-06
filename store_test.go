package dynamodbstore

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

func TestLifecycle(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(64)
	blockKey := securecookie.GenerateRandomKey(32)
	codec := securecookie.New(hashKey, blockKey)
	name := "blah"
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		return
	}

	store, err := New(tableName, codec)
	if err != nil {
		t.Errorf("expected nil; got %v", err)
		return
	}

	// New Session ------------------------

	req, _ := http.NewRequest("GET", "http://localhost", nil)
	session, err := store.New(req, name)
	if err != nil {
		t.Errorf("expected New returns nil; got %v", err)
		return
	}
	if !session.IsNew {
		t.Error("expected new session")
		return
	}

	// Save -------------------------------

	w := httptest.NewRecorder()
	err = store.Save(req, w, session)
	if err != nil {
		t.Errorf("expected Save returns nil; got %v", err)
		return
	}
	cookies := w.Result().Cookies()
	if v := len(cookies); v != 1 {
		t.Errorf("expected Save sets 1 cookie; got %v", v)
		return
	}
	cookie := cookies[0]

	// Existing Session -------------------

	req.AddCookie(cookie)
	found, err := store.New(req, name)
	if err != nil {
		t.Errorf("expected nil; got %v", err)
		return
	}
	if found.IsNew {
		t.Error("expected existing session; got new session")
		return
	}
}

func TestSerialize(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(64)
	blockKey := securecookie.GenerateRandomKey(32)
	codec := securecookie.New(hashKey, blockKey)

	name := "blah"
	store, err := New("blah", codec)
	if err != nil {
		t.Errorf("expected nil; got %v", err)
		return
	}

	options := &sessions.Options{
		Path:     "path",
		Domain:   "domain",
		MaxAge:   123,
		Secure:   true,
		HttpOnly: true,
	}
	session := &sessions.Session{
		Values: map[interface{}]interface{}{
			"hello": "world",
		},
		Options: options,
	}
	av, err := store.marshal(name, session)
	if err != nil {
		t.Errorf("expected nil; got %v", err)
		return
	}

	restored := &sessions.Session{}
	err = store.unmarshal(name, av, restored)
	if err != nil {
		t.Errorf("expected nil; got %v", err)
		return
	}

	if session.Values["hello"] != "world" {
		t.Errorf("expected hello:world; got %#v\n", session.Values)
		return
	}

	if !reflect.DeepEqual(options, session.Options) {
		t.Errorf("expected %#v; got %#v", options, session.Options)
		return
	}
}
