# dynastore

AWS DynamoDB store for Gorilla Toolkit using AWS library.  

This is a fork and overhaul of the original repo which hadn't been updated in a long time.  I was able to simplify it quite a bit with the v2 API and remove all the serialization code. It also no longer produces output when it encounters errors, it just returns them.

Uses the official AWS library, [github.com/aws/aws-sdk-go-v2/aws](github.com/aws/aws-sdk-go-v2/aws)

### Installation

```
go get github.com/andrewwatson/dynastore@v0.1.0 (current latest tag)
```

### Environment Variables

dynastore uses the common AWS environment variables:

* ```AWS_DEFAULT_REGION``` or ```AWS_REGION```
* ```AWS_ACCESS_KEY_ID``` 
* ```AWS_SECRET_ACCESS_KEY```
 
dynastore will also use AWS roles if they are available.  In that case, only
```AWS_DEFAULT_REGION``` or ```AWS_REGION``` need be set.

Alternately, AWS settings can be specified using Options:

* ```dynastore.AWSConfig(*aws.Config)``` 
* ```dynastore.DynamoDB(*dynamodb.DynamoDB)```

### Tables

dynastore provides a utility to create/delete the dynamodb table.

#### Create Table

```
dynastore -table your-table-name -read 5 -write 5 
```

#### Delete Table

Use the -delete flag to indicate the tables should be deleted instead.

```
dynastore -table your-table-name -delete 
```

## Example

```go
// Create Store
store, err := dynastore.New("session", dynastore.Path("/"), dynastore.HTTPOnly())
if err != nil {
  log.Fatalln(err)
}

// Get Session
session, err := store.Get(req, "session-key")
if err != nil {
  log.Fatalln(err)
}

// Add a Value
session.Values["hello"] = "world"

// Save Session
err := session.Save(req, w)
if err != nil {
  log.Fatalln(err)
}

// Delete Session
session.Options.MaxAge = -1
err := session.Save(req, w)
if err != nil {
  log.Fatalln(err)
}
```