// Copyright 2017 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package dynastore_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/andrewwatson/dynastore"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

func SetupStore() (*dynastore.Store, error) {

	tableName := "session"

	cfg, err := config.LoadDefaultConfig(context.TODO(), func(o *config.LoadOptions) error {
		o.Region = "us-east-1"
		return nil
	})
	if err != nil {
		return nil, err
	}

	svc := dynamodb.NewFromConfig(cfg)
	store, err := dynastore.New(tableName, svc)

	return store, err
}
func TestSaveData(t *testing.T) {

	ctx := context.TODO()

	store, err := SetupStore()
	if err != nil {
		t.Error(err)
	}

	session := sessions.Session{}
	session.ID = uuid.NewString()

	t.Logf("Session ID %s\n", session.ID)

	session.Values = map[interface{}]interface{}{
		"test": "one",
	}
	err = store.Persist(ctx, "session", &session)

	if err != nil {
		t.Error(err)
	}

	err = store.Delete(ctx, session.ID)
	if err != nil {
		t.Error(err)
	}

}

func TestLoadData(t *testing.T) {
	ctx := context.TODO()

	store, err := SetupStore()
	if err != nil {
		t.Error(err)
	}
	session := sessions.Session{}
	session.ID = "6bbb5213-a203-4758-82c0-ab7cbd9ea157"

	err = store.Load(ctx, session.ID, &session)
	if err != nil {
		t.Error(err)
	}
	t.Logf("Values %#v\n", session.Values)
}

func TestSaveRequest(t *testing.T) {

	store, err := SetupStore()
	if err != nil {
		t.Error(err)
	}

	req, _ := http.NewRequest(http.MethodGet, "www.coxautoinc.com", nil)
	session, err := store.New(req, "cookie")
	if err != nil {
		t.Error(err)
	}
	if !session.IsNew {
		t.Fail()
	}
	session.Values["test"] = "test"
	session.Save(req, FakeResponseWriter{})

}

type FakeResponseWriter struct{}

func (f FakeResponseWriter) Header() http.Header {
	return http.Header{}
}

func (f FakeResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (f FakeResponseWriter) WriteHeader(int) {
	return
}

// func TestLifecycle(t *testing.T) {
// 	hashKey := securecookie.GenerateRandomKey(64)
// 	blockKey := securecookie.GenerateRandomKey(32)
// 	codec := securecookie.New(hashKey, blockKey)
// 	name := "blah"
// 	tableName := os.Getenv("TABLE_NAME")
// 	if tableName == "" {
// 		return
// 	}

// 	testCases := map[string][]dynastore.Option{
// 		"gob":   {dynastore.TableName(tableName)},
// 		"codec": {dynastore.TableName(tableName), dynastore.Codecs(codec)},
// 	}

// 	for label, tc := range testCases {
// 		t.Run(label, func(t *testing.T) {
// 			store, err := dynastore.New(tableName, tc...)
// 			if err != nil {
// 				t.Errorf("expected nil; got %v", err)
// 				return
// 			}

// 			// New Session ------------------------

// 			req, _ := http.NewRequest("GET", "http://localhost", nil)
// 			session, err := store.New(req, name)
// 			if err != nil {
// 				t.Errorf("expected New returns nil; got %v", err)
// 				return
// 			}
// 			if !session.IsNew {
// 				t.Error("expected new session")
// 				return
// 			}

// 			// Save -------------------------------

// 			w := httptest.NewRecorder()
// 			err = store.Save(req, w, session)
// 			if err != nil {
// 				t.Errorf("expected Save returns nil; got %v", err)
// 				return
// 			}
// 			cookies := w.Result().Cookies()
// 			if v := len(cookies); v != 1 {
// 				t.Errorf("expected Save sets 1 cookie; got %v", v)
// 				return
// 			}
// 			cookie := cookies[0]

// 			// Existing Session -------------------

// 			req.AddCookie(cookie)
// 			found, err := store.New(req, name)
// 			if err != nil {
// 				t.Errorf("expected nil; got %v", err)
// 				return
// 			}
// 			if found.IsNew {
// 				t.Error("expected existing session; got new session")
// 				return
// 			}

// 			// Delete Session ---------------------

// 			found.Options.MaxAge = -1
// 			w = httptest.NewRecorder()
// 			err = store.Save(req, w, found)
// 			if err != nil {
// 				t.Errorf("expected Save returns nil; got %v", err)
// 				return
// 			}
// 			cookies = w.Result().Cookies()
// 			if v := len(cookies); v != 1 {
// 				t.Errorf("expected Save sets 1 cookie; got %v", v)
// 				return
// 			}
// 			if cookie := cookies[0]; cookie.Value != "" {
// 				t.Errorf("expected cookie to be cleared; got %v", cookie.Value)
// 				return
// 			}

// 			// Verify Session Deleted -------------

// 			found, err = store.New(req, name)
// 			if err != nil {
// 				t.Errorf("expected nil; got %v", err)
// 				return
// 			}
// 			if !found.IsNew {
// 				t.Error("expected new session; got existing session")
// 				return
// 			}
// 		})
// 	}
// }
