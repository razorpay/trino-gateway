----------------------------------------------------<!-- markdownlint-capture -->
##### Observations
Direct connectivity

NOTE: for curl the syntax is `query` instead of `"query"`  or `{query}` or `{"query"}`


python - y
go - busted -- throws a semantic error on trino side
curl - works

Via gateway
python - works
go - busted
curl - works
trino go client sends full queries as prepared statement
i.e.
for running `Select 1`
it will send `EXECUTE _trino_go USING 1`
with header `X-Trino-Prepared-Statement: _trino_go=SELECT+1`


-----
????? need to verify, trino doesnt start executing query unless the nextUri is hit first time

------------------ 
Swagger
swaggerui/index.html - contains location of loading swagger json file
to generate swagger to proto mapping
go run github.com/go-bridget/twirp-swagger-gen -in rpc/gateway/service.proto -out swaggerui/service.swagger.json -host "localhost:8000"

protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto

##### Reasonings for reverse proxy flow

elazarl/goproxy
is a forward proxy

coreos/goproxy
is fork of previous one with reverse proxy support BUT 

requests sent by all trino clients + normal curl requests
have
`http.Request.URL.Host` & `http.Request.URL.Scheme` set as nil
so goproxy cant work 
https://github.com/coreos/goproxy/blob/f8dc2d7ba04e38fa53c1a09be5fa92c62db6af62/proxy.go#L113



solution:
inherit its type & override request URL

-- not sure how to do it in go

OR 

fork it and remove this restriction

-- need to check how its built

OR 
implement own basic reverse Proxy handler

-- lot of handling around request parsing, json marshaling unmarshaling etc.




## Setup ENV
Golang
Twirp
https://twitchtv.github.io/twirp/docs/install.html

gopherJs


Setup a MySQL DB instance


## Build
go mod download
protoc --go_out=. --twirp_out=. ./rpc/gateway/service.proto
protoc --go_out=. --twirp_swagger_out=./swaggerui/ ./rpc/gateway/service.proto