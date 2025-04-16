# Go-Keycloak

A Go Fiber API with Cassandra and Keycloak integration.

## Features
- User CRUD with Cassandra
- Keycloak authentication and user management
- Dockerized setup for local development

## Prerequisites
- Docker & Docker Compose

## Quick Start

1. Clone the repository:
   ```sh
   git clone <repo-url>
   cd go-keycloack
   ```
2. Copy and edit the `.env` file as needed.
3. Start the services:
   ```sh
   docker-compose up --build
   ```
4. The API will be available at `http://localhost:3000`

## Environment Variables
- `CASSANDRA_HOST` (default: 127.0.0.1)
- `CASSANDRA_KEYSPACE` (default: testkeyspace)
- `KEYCLOAK_BASE_URL`, `REALM`, `CLIENT_ID`, `CLIENT_SECRET` (for Keycloak)

## Example Endpoints
- `POST /login` — User login via Keycloak
- `POST /users` — Create user (Keycloak + Cassandra)
- `GET /users/:id` — Get user by ID
- `PUT /users/:id` — Update user
- `DELETE /users/:id` — Delete user

## License
MIT
