package dynamodbstore

import (
	"encoding/base32"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

const (
	DefaultTableName = "dynamodbstore"
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
	errDecodeFailed     = errors.New("failed to dencode data")
)

type Store struct {
	tableName string
	codecs    []securecookie.Codec
	ddb       *dynamodb.DynamoDB
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
	return session, nil
}

// Save should persist session to the underlying store implementation.
func (s *Store) Save(req *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	err := s.save(session.Name(), session)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:  session.Name(),
		Value: session.ID,
	}

	if opts := session.Options; opts != nil {
		cookie.Path = opts.Path
		cookie.Domain = opts.Domain
		cookie.MaxAge = opts.MaxAge
		cookie.HttpOnly = opts.HttpOnly
		cookie.Secure = opts.Secure
	}

	http.SetCookie(w, cookie)
	return nil
}

func New(tableName string, codecs ...securecookie.Codec) (*Store, error) {
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}

	cfg := &aws.Config{Region: aws.String(region)}
	s, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	api := dynamodb.New(s)

	return &Store{
		tableName: tableName,
		codecs:    codecs,
		ddb:       api,
	}, nil
}

func (s *Store) save(name string, session *sessions.Session) error {
	av, err := s.marshal(name, session)
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

	err = s.unmarshal(name, out.Item, session)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) unmarshal(name string, in map[string]*dynamodb.AttributeValue, session *sessions.Session) error {
	if len(in) == 0 {
		return errNotFound
	}

	// id
	av, ok := in[idField]
	if !ok || av.S == nil {
		return errMalformedSession
	}
	id := *av.S

	// payload

	av, ok = in[valuesField]
	if !ok || av.S == nil {
		return errMalformedSession
	}

	values := map[interface{}]interface{}{}
	err := securecookie.DecodeMulti(name, *av.S, &values, s.codecs...)
	if err != nil {
		return errDecodeFailed
	}

	session.IsNew = false
	session.ID = id
	session.Values = values

	// options

	av, ok = in[optionsField]
	if ok {
		options := &sessions.Options{}
		err = dynamodbattribute.Unmarshal(av, options)
		if err != nil {
			return err
		}
		session.Options = options
	}

	return nil
}

func (s *Store) marshal(name string, session *sessions.Session) (map[string]*dynamodb.AttributeValue, error) {
	values, err := securecookie.EncodeMulti(name, session.Values, s.codecs...)
	if err != nil {
		return nil, errEncodeFailed
	}

	av := map[string]*dynamodb.AttributeValue{
		idField:     {S: aws.String(session.ID)},
		valuesField: {S: aws.String(values)},
	}

	if session.Options != nil {
		options, err := dynamodbattribute.Marshal(session.Options)
		if err != nil {
			return nil, errors.New("options failed")
		}
		av[optionsField] = options
	}

	return av, nil
}
