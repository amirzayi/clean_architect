# Go Clean Architecture Project

## What is Clean Architecture?

Clean Architecture is a software design philosophy that emphasizes separation of concerns and independence of business logic from implementation details. It was popularized by Robert C. Martin (Uncle Bob) and organizes the system into concentric layers with the following key principles:

1. **Independent of Frameworks**: The architecture doesn't depend on the existence of some library or framework
2. **Testable**: Business rules can be tested without UI, database, web server, etc.
3. **Independent of UI**: The UI can change easily without changing the rest of the system
4. **Independent of Database**: You can swap out Oracle or SQL Server for MongoDB, etc.
5. **Independent of any external agency**: Your business rules don't know anything about the outside world

## Why Use Clean Architecture?

Clean Architecture provides several important benefits:

- **High Cohesion**: Related functionality is kept close together in the same layer
- **Low Coupling**: Layers communicate through well-defined interfaces, making components easily replaceable
- **Maintainability**: Changes in one layer (like database) don't affect others
- **Testability**: Business logic can be tested without infrastructure concerns
- **Flexibility**: Easy to adapt to new requirements or technology changes

## Project Structure

The project follows a typical Clean Architecture organization with these layers:

```
project/
├── api/
│   ├── grpc/               # grpc service handlers
│   ├── http/
│   │   ├── handler/        # http handlers
│   │   ├── middleware/     # http middlewares
│   ├── proto/              # protobuf schema definition
├── cmd/                    # Main application entry point and dependency injection
│   ├── app/                # web application
│   ├── cli/                # cli application
├── internal/
│   ├── delivery/           # Interface adapters (HTTP handlers, gRPC, etc.)
│   ├── domain/             # Enterprise business rules
│   ├── service/            # Application business rules
│   ├── repository/         # Interface definitions for data access
├── pkg/                    # Concrete implementation reusable libraries
│   └── auth/               # identify user
│   └── bus/                # message broker
│   └── cache/              # cache
│   └── config/             # app configuration
│   └── errs/               # human readable internal error conversion
│   └── hash/               # hashing password
│   └── interceptor/        # grpc interceptors
│   └── jsonutil/           # json utilities
│   └── logger/             # log
│   └── mongoutil/          # mongo utilities(paginate query builder,...)
│   └── paginate/           # pagination, sorting, filter
│   └── scheduler/          # persistent task scheduler
│   └── server/
│   │   └── grpc/           # grpc server manager
│   │   └── http/           # http server manager
│   └── sqlutil/            # sql utilities(paginate query builder,...)
│   └── synq/               # sync cache via crud operations

```

## Drivers and Plugins

This project supports multiple implementations (drivers) for various components:

### Repository Drivers
- **MongoDB**
- **MySQL, Postgresql, Sqlite**
- **In-memory**

### Caching Drivers
- **Redis**
- **Memcached**
- **In-Memory**

### Message Broker Drivers
- **RabbitMQ**
- **Redis**
- **In-Memory**
- **Nats**

### Task Scheduler
- **Redis**
- **MySQL, Postgresql, Sqlite**
- **File**

### Auth
- **JWT**
- **Paseto**

### Logger
- **File**
- **External Web Service**

### Pagination, Sorting, Filters
paginate, sort, filter incoming list requests for sql and MongoDB

### Error conversion
convert internal errors to human readable response

## Getting Started

### Prerequisites
- Go 1.23+
- Docker (for running some database/broker implementations)

### Installation
1. Clone the repository
2. Install dependencies:
   ```sh
   go mod download
   ```
3. Copy the example config file and modify as needed:
   ```sh
   cp config.yaml.example config.yaml
   ```

## Run application
### webserver
```sh
go run ./cmd/app
```
### cli
```sh
go run ./cmd/cli
```

## Configuration

The application is configured via `config.[yaml,json,toml]`. You can specify which drivers to use for each component:

```yaml
db:
  driver: "mysql" # or "sqlite", "postgres", "mongodb"
  ip: 127.0.0.1
  port: 3306
  userName: amir
  password: mirzaei
  name: db

cache:
  driver: "memcached" # or "redis", "inmemory"
  ip: 127.0.0.1
  port: 11211
  userName: amir
  password: mirzaei
  prefix: app

event:
  driver: "rabbitmq" # or "redis", "memory", "nats"
  ip: 127.0.0.1
  port: 1567
  userName: amir
  password: mirzaei

scheduler:
  driver: redis # or "sqlite" or "file"
  ip: 127.0.0.1
  port: 6379
  prefix: task_scheduler
  userName: amir
  password: mirzaei

logger:
  level: 0 # debug: -1, info: 0, warn: 1, error: 2
  directory: log # store logs on specified directory
  fileCreationMode: 1 # 0: won't store in files, 1: keep in single created file, 2: keep in seperated hourly created files, 3: keep in seperated daily created files
  remoteURL: http://127.0.0.1:8080/ping # send logs to http address
  console: true # print on stdout
```

## Testing

To run tests:
```sh
go test ./...
```

The architecture makes it easy to test components in isolation by using mock implementations of interfaces.

## Dependency Injection

The application uses dependency injection to provide concrete implementations to the use cases. This is configured in `cmd/app/main.go` where you can switch between different drivers.

## Contributing

When adding new drivers or functionality:
1. Define interfaces in the appropriate layer (repository, cache, etc.)
2. Implement the interface in the driver package
3. Register the new driver in the dependency injection setup

## License

[MIT License](LICENSE)
### you need config.json file to run project. for example:
```go run cmd/web/main.go -config=/path/to/config_file.json```

### docker repository [url](https://hub.docker.com/r/92276992/clean_architect)

#### you could run docker image easily via command for example:
```sh
docker pull 92276992/clean_architect
docker run --rm -p 8070:8070 -p 8071:8071 -it --name my_container clean_architect:1.0
```
