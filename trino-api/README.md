# Trino API Module

## Overview

This module exposes an API to interact with Trino. It retrieves data using the Trino Go client, processes it, and returns it in JSON format to the client.

### Features:
- **Trino Integration:** Connects to a Trino instance and executes SQL queries.
- **Data Processing:** Applies business logic to the data before sending it to the client.
- **JSON API:** Communicates with clients using JSON format.
- **Request/Response Limitation:** Throws an error if response data exceeds 5,000 records.
- **Configurable:** Uses TOML files for different environments (development, staging, production).

## Project Structure

```plaintext
trino-api/
│
├── build/
│   ├── docker/
│       ├── dev/                   
│       │   ├── Dockerfile.api       
│       ├── entrypoint.sh  
│
├── cmd/
│   ├── api/
│       ├── main.go                  
│
├── config/
│   ├── default.toml                
│   ├── dev_docker.toml              
│   ├── stage.toml                   
│   ├── prod.toml                    
│
├── deployment/
│   ├── dev/
│       ├── docker-compose.yml       
│
├── internal/
│   ├── app/
│   │   ├── handler/
│   │   │    ├── handler.go          
│   │   │    ├── handler_test.go 
│   │   ├── process/
│   │   │    ├── processor.go        
│   │   ├── routes/
│   │   │    ├── routes.go           
│   │   ├── app.go                   
│   │
│   ├── config/
│   │   ├── config.go                
│   ├── model/
│   │   ├── model.go                 
│   ├── utils/
│   │   ├── response.go              
│   ├── services/
│   │   ├── trino/
│   │        ├── client.go           
│   └── boot/
│       ├── boot.go                  
│
├── pkg/
│   ├── config/
│       ├── config.go                
│       ├── config_test.go 
│
├── go.mod
│
├── go.sum
│
├── Makefile
│
├── README.md
│
