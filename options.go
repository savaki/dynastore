package dynamodbstore

type Option interface {
	apply(*Store)
}
