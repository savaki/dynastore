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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gorilla/sessions"
)

// Option provides options to creating a dynastore
type Option func(*Store)

// DynamoDB allows a pre-configured dynamodb client to be supplied
func DynamoDB(ddb *dynamodb.Client) Option {
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

// Output
// func Output(w io.Writer) Option {
// 	return func(s *Store) {
// 		s.printf = func(format string, args ...interface{}) {
// 			fmt.Fprintf(w, format, args...)
// 		}
// 	}
// }

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
