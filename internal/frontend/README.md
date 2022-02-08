### Notes
Gopherjs compiles to js to run entirely in client browser
unlike java nashorn or similar engine in jvm which allows running js code
we dont hav that luxury,
as a result the gatewayserver API and UI need to reside under same endpoint or else
due to CORS https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSMissingAllowOrigin
browsers won't allow connectivity between both.
This also solves the problem of maintaining state for UI as its completely client side

### Running in dev mode for testing

The easiest way to run the examples as WebAssembly is via [`wasmserve`](https://github.com/hajimehoshi/wasmserve).

Install it (**using Go 1.14+**):

```bash
go get -u github.com/hajimehoshi/wasmserve
```

Then run an example:

```bash
cd trino-gateway/internal/frontend
wasmserve
```

Then navigate to http://localhost:8080/