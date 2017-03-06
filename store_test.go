package dynastore

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/securecookie"
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

	testCases := map[string][]Option{
		"gob":   {TableName(tableName)},
		"codec": {TableName(tableName), Codecs(codec)},
	}

	for label, tc := range testCases {
		t.Run(label, func(t *testing.T) {
			store, err := New(tc...)
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

			// Delete Session ---------------------

			found.Options.MaxAge = -1
			w = httptest.NewRecorder()
			err = store.Save(req, w, found)
			if err != nil {
				t.Errorf("expected Save returns nil; got %v", err)
				return
			}
			cookies = w.Result().Cookies()
			if v := len(cookies); v != 1 {
				t.Errorf("expected Save sets 1 cookie; got %v", v)
				return
			}
			if cookie := cookies[0]; cookie.Value != "" {
				t.Errorf("expected cookie to be cleared; got %v", cookie.Value)
				return
			}

			// Verify Session Deleted -------------

			found, err = store.New(req, name)
			if err != nil {
				t.Errorf("expected nil; got %v", err)
				return
			}
			if !found.IsNew {
				t.Error("expected new session; got existing session")
				return
			}
		})
	}
}
