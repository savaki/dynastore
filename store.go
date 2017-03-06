package dynastore

import (
	"encoding/base32"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

const (
	// DefaultTableName is the default table name used by the dynamodb store
	DefaultTableName = "dynastore"
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
	codecs     []securecookie.Codec
	config     *aws.Config
	ddb        *dynamodb.DynamoDB
	serializer serializer
	options    sessions.Options
}

// Get should return a cached session.
func (s *Store) Get(req *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(req).Get(s, name)
}

// New should create and return a new session.
//
// Note that New should never return a nil session, even in the case of
// an error if using the Registry infrastructure to cache the session.
func (s *Store) New(req *http.Request, name string) (*sessions.Session, error) {
	if cookie, errCookie := req.Cookie(name); errCookie == nil {
		session := sessions.NewSession(s, name)
		err := s.load(name, cookie.Value, session)
		if err == nil {
			return session, nil
		}
		if err != errNotFound {
			return nil, err
		}
	}

	session := sessions.NewSession(s, name)
	session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	session.IsNew = true
	session.Options = &sessions.Options{
		Path:     s.options.Path,
		Domain:   s.options.Domain,
		MaxAge:   s.options.MaxAge,
		Secure:   s.options.Secure,
		HttpOnly: s.options.HttpOnly,
	}

	return session, nil
}

// Save should persist session to the underlying store implementation.
func (s *Store) Save(req *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	err := s.save(session.Name(), session)
	if err != nil {
		return err
	}

	if session.Options != nil && session.Options.MaxAge < 0 {
		cookie := newCookie(session, session.Name(), "")
		http.SetCookie(w, cookie)
		return s.delete(session.ID)
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
func New(opts ...Option) (*Store, error) {
	store := &Store{
		tableName: DefaultTableName,
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

func (s *Store) save(name string, session *sessions.Session) error {
	av, err := s.serializer.marshal(name, session)
	if err != nil {
		return err
	}

	_, err = s.ddb.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) delete(id string) error {
	_, err := s.ddb.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
	})
	return err
}

// load loads a session data from the database.
// True is returned if there is a session data in the database.
func (s *Store) load(name, value string, session *sessions.Session) error {
	out, err := s.ddb.GetItem(&dynamodb.GetItemInput{
		TableName:      aws.String(s.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(value)},
		},
	})
	if err != nil {
		return err
	}

	err = s.serializer.unmarshal(name, out.Item, session)
	if err != nil {
		return err
	}

	return nil
}

type serializer interface {
	marshal(name string, session *sessions.Session) (map[string]*dynamodb.AttributeValue, error)
	unmarshal(name string, in map[string]*dynamodb.AttributeValue, session *sessions.Session) error
}
