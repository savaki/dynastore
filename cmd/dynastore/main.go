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
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/savaki/dynastore"
)

func main() {
	var (
		tableName     = flag.String("table", dynastore.DefaultTableName, "DynamoDB table name")
		delete        = flag.Bool("delete", false, "Delete the table")
		readCapacity  = flag.Int64("read", 5, "Provisioned DynamoDB Read capacity")
		writeCapacity = flag.Int64("write", 5, "Provisioned DynamoDB Write capacity")
	)
	flag.Parse()

	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}

	cfg := &aws.Config{Region: aws.String(region)}
	s, err := session.NewSession(cfg)
	if err != nil {
		log.Fatalf("Unable to create AWS session - %v\n", err)
	}

	api := dynamodb.New(s)
	if *delete {
		fmt.Printf("Deleting dynamodb table, %v [%v]\n", *tableName, region)
		_, err := api.DeleteTable(&dynamodb.DeleteTableInput{
			TableName: tableName,
		})
		if err != nil {
			fmt.Printf("** ERR *** unable to delete dynamodb table - %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully deleted table")

	} else {
		fmt.Printf("Creating dynamodb table, %v [%v]\n", *tableName, region)
		_, err := api.CreateTable(&dynamodb.CreateTableInput{
			TableName: tableName,
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("id"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  readCapacity,
				WriteCapacityUnits: writeCapacity,
			},
		})
		if err != nil {
			if v, ok := err.(awserr.Error); ok {
				if v.Code() == "ResourceInUseException" {
					fmt.Println("Table already exists")
					return
				}
			}
			fmt.Printf("** ERR *** unable to create dynamodb table - %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully created table")
	}
}
