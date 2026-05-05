# Framework Detection Patterns

Detect the framework by scanning the project structure and apply the correct route extraction strategy.

## Go

Look for route registrations in `cmd/`, `main.go`, `routes/`, `router/`, `api/`, `handlers/`.

### Echo
```go
e.POST("/v1/orders", handler.CreateOrder)
e.GET("/v1/orders/:id", handler.GetOrder)
g := e.Group("/v1/merchants", authMiddleware)
```

### Chi
```go
r.Post("/v1/orders", handler.CreateOrder)
r.Get("/v1/orders/{id}", handler.GetOrder)
r.Group(func(r chi.Router) {
    r.Use(authMiddleware)
    r.Get("/v1/merchants", handler.ListMerchants)
})
```

### Gin
```go
router.POST("/v1/orders", handler.CreateOrder)
router.GET("/v1/orders/:id", handler.GetOrder)
auth := router.Group("/v1/merchants", authMiddleware)
```

### Gorilla Mux
```go
r.HandleFunc("/v1/orders", handler.CreateOrder).Methods("POST")
r.HandleFunc("/v1/orders/{id}", handler.GetOrder).Methods("GET")
```

### Twirp (Protobuf)
```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (Order);
}
```

### net/http
```go
http.HandleFunc("/v1/orders", handler.CreateOrder)
http.Handle("/v1/orders/", handler.OrderHandler())
```

## Laravel / PHP

Look for `routes/api.php`, `routes/web.php`, `app/Http/Controllers/`.

### Route Definitions
```php
Route::get('/v1/payments', 'PaymentController@index');
Route::post('/v1/payments', 'PaymentController@store');
Route::get('/v1/payments/{id}', 'PaymentController@show');
Route::put('/v1/payments/{id}', 'PaymentController@update');
```

### Resource Routes
```php
Route::resource('payments', 'PaymentController');
// Generates: index, store, show, update, destroy
```

### Route Groups with Middleware
```php
Route::group(['middleware' => ['auth:private']], function () {
    Route::post('/v1/orders', 'OrderController@store');
});

Route::group(['middleware' => ['auth:public']], function () {
    Route::get('/v1/methods', 'MethodController@index');
});
```

## Express / Node.js

Look for `app.get()`, `app.post()`, `router.get()` in `routes/`, `src/`, `app.js`, `server.js`.

### App-level Routes
```javascript
app.get('/api/users', userController.list);
app.post('/api/users', authMiddleware, userController.create);
app.get('/api/users/:id', authMiddleware, userController.get);
```

### Router-level Routes
```javascript
const router = express.Router();
router.get('/', userController.list);
router.post('/', userController.create);
router.get('/:id', userController.get);
app.use('/api/users', authMiddleware, router);
```

## Fastify

Look for `fastify.get()`, `fastify.post()` or route schema definitions.

### Route Definitions
```javascript
fastify.get('/api/users', { preHandler: [authMiddleware] }, handler.list);
fastify.post('/api/users', { schema: createUserSchema }, handler.create);
```

### Route Schema with Validation
```javascript
fastify.route({
  method: 'POST',
  url: '/api/users',
  schema: {
    body: { type: 'object', properties: { name: { type: 'string' } } }
  },
  handler: userController.create
});
```

## Generic Detection

If the framework is not recognized, search for HTTP method patterns:
- `GET`, `POST`, `PUT`, `DELETE`, `PATCH` in route or handler files
- URL path patterns like `/v1/`, `/api/`, `/v2/`
- Middleware or interceptor registrations near route definitions
