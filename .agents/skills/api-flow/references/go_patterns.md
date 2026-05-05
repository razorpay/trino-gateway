# Go Code Analysis Patterns

This reference provides patterns for identifying different types of operations in Go codebases.

## Route Definition Patterns

### Gin Framework
```go
// Group routes
router := gin.Default()
router.POST("/users/create", CreateUserHandler)
router.GET("/users/:id", GetUserHandler)

// Route groups
api := router.Group("/api/v1")
api.POST("/accounts", CreateAccount)
```

### Echo Framework
```go
e := echo.New()
e.POST("/users", CreateUser)
e.GET("/users/:id", GetUser)
```

### Gorilla Mux
```go
r := mux.NewRouter()
r.HandleFunc("/users", CreateUser).Methods("POST")
r.HandleFunc("/users/{id}", GetUser).Methods("GET")
```

### Chi Router
```go
r := chi.NewRouter()
r.Post("/users", CreateUser)
r.Get("/users/{id}", GetUser)
```

### Fiber
```go
app := fiber.New()
app.Post("/users", CreateUser)
app.Get("/users/:id", GetUser)
```

### Standard Library (net/http)
```go
http.HandleFunc("/users", CreateUserHandler)
http.Handle("/api/", apiHandler)
```

## Database Operation Patterns

### SQL Direct Queries
```go
// database/sql
db.Query("SELECT * FROM users WHERE id = ?", id)
db.QueryRow("SELECT name FROM users WHERE id = ?", id)
db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
db.ExecContext(ctx, "UPDATE users SET status = ? WHERE id = ?", status, id)

// Prepared statements
stmt.Query(id)
stmt.Exec(name, email)
```

### GORM
```go
// Find operations
db.Find(&users)
db.First(&user, id)
db.Where("email = ?", email).First(&user)
db.Model(&User{}).Where("status = ?", "active").Find(&users)

// Create operations
db.Create(&user)
db.Save(&user)

// Update operations
db.Model(&user).Update("name", newName)
db.Model(&user).Updates(User{Name: name, Email: email})

// Delete operations
db.Delete(&user)
db.Where("age < ?", 18).Delete(&User{})

// Raw queries
db.Raw("SELECT * FROM users WHERE id = ?", id).Scan(&user)
db.Exec("UPDATE users SET status = ? WHERE id = ?", status, id)
```

### sqlx
```go
db.Get(&user, "SELECT * FROM users WHERE id = ?", id)
db.Select(&users, "SELECT * FROM users WHERE status = ?", status)
db.Exec("INSERT INTO users (name) VALUES (?)", name)
db.NamedExec("INSERT INTO users (name, email) VALUES (:name, :email)", user)
```

### Squirrel (Query Builder)
```go
sq.Select("*").From("users").Where(sq.Eq{"id": id}).RunWith(db).Query()
sq.Insert("users").Columns("name", "email").Values(name, email).RunWith(db).Exec()
sq.Update("users").Set("status", status).Where(sq.Eq{"id": id}).RunWith(db).Exec()
```

### MongoDB
```go
// mongo-go-driver
collection.Find(ctx, bson.M{"status": "active"})
collection.FindOne(ctx, bson.M{"_id": id})
collection.InsertOne(ctx, user)
collection.UpdateOne(ctx, filter, update)
collection.DeleteOne(ctx, filter)
```

### Redis (as database)
```go
// go-redis
rdb.Get(ctx, key)
rdb.Set(ctx, key, value, expiration)
rdb.HGetAll(ctx, key)
rdb.ZRange(ctx, key, 0, -1)
```

## HTTP Client Patterns (External API Calls)

### Standard Library
```go
http.Get("https://api.example.com/users")
http.Post("https://api.example.com/users", "application/json", body)
http.Do(req)

// Custom client
client := &http.Client{}
client.Do(req)
```

### Resty
```go
resty.R().Get("https://api.example.com/users")
resty.R().Post("https://api.example.com/users")
resty.R().SetBody(user).Post(url)
```

### Common HTTP client patterns
```go
// Look for these variable names and patterns
client.Do(req)
httpClient.Get(url)
restClient.Post(url, body)
apiClient.Send(req)
```

## Cache Operation Patterns

### Redis
```go
// go-redis
rdb.Get(ctx, key)
rdb.Set(ctx, key, value, expiration)
rdb.SetNX(ctx, key, value, expiration)
rdb.Del(ctx, keys...)
rdb.Exists(ctx, keys...)
rdb.HGet(ctx, key, field)
rdb.HSet(ctx, key, field, value)
rdb.Incr(ctx, key)
rdb.Expire(ctx, key, expiration)

// Pipeline
pipe := rdb.Pipeline()
pipe.Set(ctx, key, value, 0)
pipe.Exec(ctx)
```

### Memcached
```go
// gomemcache
mc.Get(key)
mc.Set(&memcache.Item{Key: key, Value: value})
mc.Delete(key)
mc.Add(&memcache.Item{Key: key, Value: value})
```

### In-memory caches
```go
// bigcache
cache.Get(key)
cache.Set(key, value)

// go-cache
cache.Get(key)
cache.Set(key, value, expiration)

// ristretto
cache.Get(key)
cache.Set(key, value, cost)
```

## Message Queue Patterns

### Kafka
```go
// sarama (Kafka client)
// Producer
producer.SendMessage(&sarama.ProducerMessage{Topic: topic, Value: value})

// Consumer
consumer.Subscribe(topics, handler)
consumer.Consume(ctx, topics, handler)

// Confluent Kafka
p.Produce(&kafka.Message{TopicPartition: tp, Value: value}, nil)
```

### RabbitMQ
```go
// amqp
ch.Publish(exchange, key, mandatory, immediate, msg)
ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
ch.QueueDeclare(name, durable, autoDelete, exclusive, noWait, args)
```

### AWS SQS
```go
// aws-sdk-go
sqs.SendMessage(&sqs.SendMessageInput{QueueUrl: queueURL, MessageBody: body})
sqs.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: queueURL})
sqs.DeleteMessage(&sqs.DeleteMessageInput{QueueUrl: queueURL, ReceiptHandle: handle})
```

### Google Pub/Sub
```go
// cloud.google.com/go/pubsub
topic.Publish(ctx, &pubsub.Message{Data: data})
sub.Receive(ctx, handler)
```

### NATS
```go
// nats.go
nc.Publish(subject, data)
nc.Subscribe(subject, handler)
nc.Request(subject, data, timeout)
```

### NSQ
```go
// go-nsq
producer.Publish(topic, message)
consumer.AddHandler(handler)
```

## Common Service/Repository Patterns

### Service Layer
```go
// Common method patterns
type UserService struct {
    repo UserRepository
    cache Cache
}

func (s *UserService) Create(ctx context.Context, user *User) error
func (s *UserService) GetByID(ctx context.Context, id string) (*User, error)
func (s *UserService) Update(ctx context.Context, user *User) error
func (s *UserService) Delete(ctx context.Context, id string) error
```

### Repository Pattern
```go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

## Middleware Patterns

```go
// Standard middleware signature
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // middleware logic
        next.ServeHTTP(w, r)
    })
}

// Gin middleware
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // middleware logic
        c.Next()
    }
}

// Echo middleware
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // middleware logic
        return next(c)
    }
}
```

## Transaction Patterns

```go
// database/sql
tx, err := db.Begin()
tx.Exec(...)
tx.Commit() // or tx.Rollback()

// GORM
tx := db.Begin()
tx.Create(&user)
tx.Commit() // or tx.Rollback()

// sqlx
tx, err := db.Beginx()
tx.Exec(...)
tx.Commit()
```

## Goroutine/Concurrency Patterns

```go
// Basic goroutine
go functionCall()
go func() { ... }()

// WaitGroup
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // work
}()
wg.Wait()

// Channels
ch := make(chan Result)
go func() {
    ch <- result
}()
result := <-ch

// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

## External Service Client Patterns

Look for commonly named client variables and types:
- `*http.Client`
- `APIClient`, `RestClient`, `ServiceClient`
- `StripeClient`, `TwilioClient`, `SendGridClient` (payment, SMS, email services)
- `S3Client`, `BlobStorageClient` (cloud storage)
- Package imports like `"github.com/stripe/stripe-go"`, `"github.com/aws/aws-sdk-go"`

## Validation Patterns

```go
// go-validator
validate.Struct(user)
validate.Var(email, "required,email")

// ozzo-validation
validation.Validate(value, validation.Required, validation.Length(5, 20))

// Custom validation
if user.Email == "" {
    return errors.New("email is required")
}
```

## Error Handling Patterns

```go
// Standard error handling
if err != nil {
    return err
}

// Wrapped errors
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// pkg/errors
if err != nil {
    return errors.Wrap(err, "failed to create user")
}
```

## Analysis Strategy

1. **Start from Route**: Find the route definition for the given endpoint
2. **Identify Handler**: Locate the handler function referenced in the route
3. **Follow the Call Chain**: Trace through function calls in execution order
4. **Categorize Operations**: Identify database, cache, HTTP, queue operations
5. **Track Context Flow**: Note what data is passed between layers
6. **Capture Conditionals**: Record if/else branches that affect flow
7. **Note Async Operations**: Identify goroutines and async processing
8. **Include Error Paths**: Track error handling and recovery logic
