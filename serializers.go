package dynastore

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type codecSerializer struct {
	codecs []securecookie.Codec
}

func (c *codecSerializer) marshal(name string, session *sessions.Session) (map[string]*dynamodb.AttributeValue, error) {
	values, err := securecookie.EncodeMulti(name, session.Values, c.codecs...)
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
			return nil, err
		}
		av[optionsField] = options
	}

	return av, nil
}

func (c *codecSerializer) unmarshal(name string, in map[string]*dynamodb.AttributeValue, session *sessions.Session) error {
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
	err := securecookie.DecodeMulti(name, *av.S, &values, c.codecs...)
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

type gobSerializer struct {
}

func (d *gobSerializer) marshal(name string, session *sessions.Session) (map[string]*dynamodb.AttributeValue, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(session.Values)
	if err != nil {
		return nil, errEncodeFailed
	}
	values := base64.StdEncoding.EncodeToString(buf.Bytes())

	av := map[string]*dynamodb.AttributeValue{
		idField:     {S: aws.String(session.ID)},
		valuesField: {S: aws.String(values)},
	}

	// encode options

	if session.Options != nil {
		options, err := dynamodbattribute.Marshal(session.Options)
		if err != nil {
			return nil, err
		}
		av[optionsField] = options
	}

	return av, nil
}

func (d *gobSerializer) unmarshal(name string, in map[string]*dynamodb.AttributeValue, session *sessions.Session) error {
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
	if !ok && av.S != nil {
		return errMalformedSession
	}

	data, err := base64.StdEncoding.DecodeString(*av.S)
	if err != nil {
		return errDecodeFailed
	}
	values := map[interface{}]interface{}{}
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&values)
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
