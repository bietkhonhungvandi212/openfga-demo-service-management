# Open FGA Demo

## Overview

This toy project is setup services management by openfga

## Guidance setup

### Folder structure

```plaintext
servicer-management/
│
├── docker-compose.yml
├── README.md
│
├── model/
│   └── service-model.fga
│
├── service-internal/
│   ├── dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go
│
└── service-caller/
    ├── dockerfile
    ├── go.mod
    ├── go.sum
    └── main.go
```

### Architecture

```plaintext

                 +-----------------------+
                 |       OpenFGA         |
                 |  Authorization Server |
                 |  (relationship store) |
                 +-----------+-----------+
                             ^
                             |
                     Permission Check
                             |
                     +-------+--------+
                     |  Service A     |
                     |  (Internal API)|
                     |----------------|
                     | Auth Middleware|
                     +-------+--------+
                             ^
                HTTP Request |
                 +-----------+-----------+
                 |                       |
        +--------+--------+     +--------+--------+
        | Service Caller A |     | Service Caller C |
        |  (Allowed)       |     |   (Denied)       |
        +------------------+     +------------------+
```

### Set up

install OpenFGA CLI:

```bash
    brew install openfga/tap/fga
```

1. Set up OpenFGA server

```bash
    docker compose up openfga
```

2. Create Store

```bash
    fga store create --name service-auth
```

The Response:

```json
{
  "store": {
    "created_at": "2026-03-12T14:55:08.915264844Z",
    "id": "01KKH8PP3KRM77KB8DF589EJ4K",
    "name": "service-auth",
    "updated_at": "2026-03-12T14:55:08.915264844Z"
  }
}
```

3. Replace the store id in docker compose

Replace STORE_ID in docker-compose for service-internal

4. Create schema authentication

```bash
fga model write \
  --store-id=01KKH8PP3KRM77KB8DF589EJ4K \
  --file=model/service-model.fga
```

The response looks like:

```json
{
  "authorization_model_id": "01KKH8TP4C8Z9BQCBVS22K44Y4"
}
```

you can check again by this command:

```bash
fga model list --store-id=01KKH8PP3KRM77KB8DF589EJ4K
```

The response looks like:

```json
{
  "authorization_models": [
    {
      "id": "01KKH8TP4C8Z9BQCBVS22K44Y4",
      "created_at": "2026-03-12T14:57:20.012Z"
    }
  ]
}
```

5.Define permission

```bash
fga tuple write service:service-caller-a can_call service:service-internal-a \
  --store-id=01KKH8PP3KRM77KB8DF589EJ4K
```

the response looks like:

```json
{
  "successful": [
    {
      "object": "service:service-internal-a",
      "relation": "can_call",
      "user": "service:service-caller-a"
    }
  ]
}
```

6. Start internal + caller services

```bash
docker compose up
```

7. Now test permission

- Calling to servicer caller A, so this service will call to internal service. The expectation that we will receive the response without forbidden

```bash
curl 'http://localhost:8083/internal'
```

the response will be:

```json
{ "response": "ok", "service": "service-internal" }
```

- Calling to servicer caller B, so this service will call to internal service. The expectation that we will be forbiddened

```bash
curl 'http://localhost:8084/internal'
```

the response will be:

```json
{ "response": "{\"error\":\"Forbidden\"}ok", "service": "service-internal" }
```
