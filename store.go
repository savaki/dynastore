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
package dynastore

import (
	"context"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

const (

	// DefaultTTLField contains the default name of the ttl field
	DefaultTTLField = "ttl"
)

const (
	idField      = "id"
	valuesField  = "values"
	optionsField = "options"
)

var (
	errNotFound         = errors.New("session not found")
	errMalformedSession = errors.New("malformed session data")
	errEncodeFailed     = errors.New("failed to encode data")
	errDecodeFailed     = errors.New("failed to decode data")
)

// Store provides an implementation of the gorilla sessions.Store interface backed by DynamoDB
type Store struct {
	tableName string
	ddb       *dynamodb.Client
	options   sessions.Options
}

// Get should return a cached session.
func (store *Store) Get(req *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(req).Get(store, name)
}

// New should create and return a new session.
//
// Note that New should never return a nil session, even in the case of
// an error if using the Registry infrastructure to cache the session.
func (store *Store) New(req *http.Request, name string) (*sessions.Session, error) {
	if cookie, errCookie := req.Cookie(name); errCookie == nil {
		s := sessions.NewSession(store, name)
		err := store.Load(req.Context(), cookie.Value, s)
		if err == nil {
			return s, nil
		}
	}

	s := sessions.NewSession(store, name)
	s.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	s.IsNew = true
	s.Options = &sessions.Options{
		Path:     store.options.Path,
		Domain:   store.options.Domain,
		MaxAge:   store.options.MaxAge,
		Secure:   store.options.Secure,
		HttpOnly: store.options.HttpOnly,
	}

	return s, nil
}

// Save should persist session to the underlying store implementation.
func (store *Store) Save(req *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	err := store.Persist(req.Context(), session.Name(), session)
	if err != nil {
		return err
	}

	if session.Options != nil && session.Options.MaxAge < 0 {
		cookie := newCookie(session, session.Name(), "")
		http.SetCookie(w, cookie)
		return store.Delete(req.Context(), session.ID)
	}

	if !session.IsNew {
		// no need to set cookies if they already exist
		return nil
	}

	cookie := newCookie(session, session.Name(), session.ID)
	http.SetCookie(w, cookie)
	return nil
}

func newCookie(session *sessions.Session, name, value string) *http.Cookie {
	cookie := &http.Cookie{
		Name:  name,
		Value: value,
	}

	if opts := session.Options; opts != nil {
		cookie.Path = opts.Path
		cookie.Domain = opts.Domain
		cookie.MaxAge = opts.MaxAge
		cookie.HttpOnly = opts.HttpOnly
		cookie.Secure = opts.Secure
	}

	return cookie
}

// New instantiates a new Store that implements gorilla's sessions.Store interface
func New(tablename string, client *dynamodb.Client, opts ...Option) (*Store, error) {
	store := &Store{
		ddb:       client,
		tableName: tablename,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store, nil
}

func (store *Store) Persist(ctx context.Context, name string, session *sessions.Session) error {

	session.Values["SessionHashKey"] = session.ID

	itemMap := make(map[string]types.AttributeValue)
	for i, v := range session.Values {
		itemMap[i.(string)] = &types.AttributeValueMemberS{Value: v.(string)}
	}

	_, err := store.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(store.tableName),
		Item:      itemMap,
	})

	return err
}

func (store *Store) Delete(ctx context.Context, id string) error {

	_, err := store.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(store.tableName),
		Key: map[string]types.AttributeValue{
			"SessionHashKey": &types.AttributeValueMemberS{Value: id},
		},
	})

	return err
}

// load loads a session data from the database.
// True is returned if there is a session data in the database.
func (store *Store) Load(ctx context.Context, value string, session *sessions.Session) error {

	out, err := store.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(store.tableName),
		Key: map[string]types.AttributeValue{
			"SessionHashKey": &types.AttributeValueMemberS{Value: value},
		},
	})

	for i, v := range out.Item {
		session.Values[i] = v
	}
	return err
}
