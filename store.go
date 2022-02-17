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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
	tableName  string
	ttlField   string
	codecs     []securecookie.Codec
	config     *aws.Config
	ddb        *dynamodb.DynamoDB
	serializer serializer
	options    sessions.Options
	printf     func(format string, args ...interface{})
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
		err := store.load(req.Context(), name, cookie.Value, s)
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
	err := store.save(req.Context(), session.Name(), session)
	if err != nil {
		return err
	}

	if session.Options != nil && session.Options.MaxAge < 0 {
		cookie := newCookie(session, session.Name(), "")
		http.SetCookie(w, cookie)
		return store.delete(req.Context(), session.ID)
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
func New(tablename string, opts ...Option) (*Store, error) {
	store := &Store{
		tableName: tablename,
		ttlField:  DefaultTTLField,
		printf:    func(format string, args ...interface{}) {},
	}

	for _, opt := range opts {
		opt(store)
	}

	if store.ddb == nil {
		if store.config == nil {
			region := os.Getenv("AWS_DEFAULT_REGION")
			if region == "" {
				region = os.Getenv("AWS_REGION")
			}

			store.config = &aws.Config{Region: aws.String(region)}
		}

		s, err := session.NewSession(store.config)
		if err != nil {
			return nil, err
		}

		store.ddb = dynamodb.New(s)
	}

	if len(store.codecs) > 0 {
		store.serializer = &codecSerializer{codecs: store.codecs}
	} else {
		store.serializer = &gobSerializer{}
	}

	return store, nil
}

func (store *Store) save(ctx context.Context, name string, session *sessions.Session) error {
	av, err := store.serializer.marshal(name, session)
	if err != nil {
		store.printf("dynastore: failed to marshal session - %v\n", err)
		return err
	}

	if store.ttlField != "" && session.Options != nil && session.Options.MaxAge > 0 {
		expiresAt := time.Now().Add(time.Duration(session.Options.MaxAge) * time.Second)
		ttl := strconv.FormatInt(expiresAt.Unix(), 10)
		av[store.ttlField] = &dynamodb.AttributeValue{N: aws.String(ttl)}
	}

	_, err = store.ddb.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(store.tableName),
		Item:      av,
	})
	if err != nil {
		store.printf("dynastore: PutItem failed - %v\n", err)
		return err
	}

	return nil
}

func (store *Store) delete(ctx context.Context, id string) error {
	_, err := store.ddb.DeleteItemWithContext(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(store.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
	})
	if err != nil {
		store.printf("dynastore: delete failed - %v\n", err)
		return err
	}
	return nil
}

// load loads a session data from the database.
// True is returned if there is a session data in the database.
func (store *Store) load(ctx context.Context, name, value string, session *sessions.Session) error {
	out, err := store.ddb.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(store.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(value)},
		},
	})
	if err != nil {
		store.printf("dynastore: GetItem failed\n")
		return err
	}

	if len(out.Item) == 0 {
		store.printf("dynastore: session not found\n")
		return errNotFound
	}

	ttl := int64(0)
	if av, ok := out.Item[store.ttlField]; ok {
		if av.N == nil {
			store.printf("dynastore: no ttl associated with session\n")
			return errMalformedSession
		}
		v, err := strconv.ParseInt(*av.N, 10, 64)
		if err != nil {
			store.printf("dynastore: malformed session - %v\n", err)
			return errMalformedSession
		}
		ttl = v
	}

	if ttl > 0 && ttl < time.Now().Unix() {
		store.printf("dynastore: session expired\n")
		return errNotFound
	}

	err = store.serializer.unmarshal(name, out.Item, session)
	if err != nil {
		store.printf("dynastore: unable to unmarshal session - %v\n", err)
		return err
	}

	return nil
}

type serializer interface {
	marshal(name string, session *sessions.Session) (map[string]*dynamodb.AttributeValue, error)
	unmarshal(name string, in map[string]*dynamodb.AttributeValue, session *sessions.Session) error
}
