# tm-event
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=tsel-ticketmaster_tm-event&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=tsel-ticketmaster_tm-event)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=tsel-ticketmaster_tm-event&metric=bugs)](https://sonarcloud.io/summary/new_code?id=tsel-ticketmaster_tm-event)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=tsel-ticketmaster_tm-event&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=tsel-ticketmaster_tm-event)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=tsel-ticketmaster_tm-event&metric=coverage)](https://sonarcloud.io/summary/new_code?id=tsel-ticketmaster_tm-event)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=tsel-ticketmaster_tm-event&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=tsel-ticketmaster_tm-event)


This Project is used to handle ticket master customer services including auth, registration and profile.

### Prerequisites

What things you need to install the software and how to install them

```
Golang v1.21.x
Go Mod
....
```

### Installing

A step by step series of examples that tell you have to get a development env running

Say what the step will be
- Create ENV file (.env) with this configuration:
```
APP_NAME=tm-event
APP_PORT=9800
APP_LOCATION=Asia/Jakarta
APP_DEBUG=false
CORS_ALLOWED_ORIGINS=
CORS_ALLOWED_METHOD=
OTEL_COLLECTOR_ENDPOINT=localhost:4444
```
- Then run this command (Development Issues)
```
Give the example
...
$ make run.dev
```

- Then run this command (Production Issues)
```
Give the example
...
$ make install
$ make test
$ make build
$ ./app
```

### Running the tests

Explain how to run the automated tests for this system
```sh
Give the example
...
$ make test
```

### Running the tests (With coverage appear on)

Explain how to run the automated tests for this system
```sh
Give the example
...
$ make cover
```

### Deployment

Add additional notes about how to deploy this on a live system

### Built With

* [Gorilla/Mux](https://github.com/gorilla/mux) The rest framework used
* [Mockery] Mock Up Generator
* [GoMod] - Dependency Management
* [Docker] - Container Management

### Authors

* **Patrick Maurits Sangian** - [Github](https://github.com/sangianpatrick)
