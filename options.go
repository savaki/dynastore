package dynamodbstore

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// Option provides options to creating a dynamodbstore
type Option func(*Store)

// Codecs uses the specified codecs to encrypt the cookie data
func Codecs(codecs ...securecookie.Codec) Option {
	return func(s *Store) {
		s.codecs = codecs
	}
}

// AWSConfig allows the complete AWS configuration to be specified
func AWSConfig(config *aws.Config) Option {
	return func(s *Store) {
		s.config = config
	}
}

// DynamoDB allows a pre-configured dynamodb client to be supplied
func DynamoDB(ddb *dynamodb.DynamoDB) Option {
	return func(s *Store) {
		s.ddb = ddb
	}
}

// TableName allows a custom table name to be specified
func TableName(tableName string) Option {
	return func(s *Store) {
		s.tableName = tableName
	}
}

// SessionOptions allows the default session options to be specified in a single command
func SessionOptions(options sessions.Options) Option {
	return func(s *Store) {
		s.options = options
	}
}

// Path sets the default session option of the same name
func Path(v string) Option {
	return func(s *Store) {
		s.options.Path = v
	}
}

// Domain sets the default session option of the same name
func Domain(v string) Option {
	return func(s *Store) {
		s.options.Domain = v
	}
}

// MaxAge sets the default session option of the same name
func MaxAge(v int) Option {
	return func(s *Store) {
		s.options.MaxAge = v
	}
}

// Secure sets the default session option of the same name
func Secure() Option {
	return func(s *Store) {
		s.options.Secure = true
	}
}

// HTTPOnly sets the default session option HttpOnly; HTTP is all capped to satisfy golint
func HTTPOnly() Option {
	return func(s *Store) {
		s.options.HttpOnly = true
	}
}
