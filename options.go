package dynamodbstore

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/securecookie"
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
