# Backend Development Guidelines for AI Agents (AGENTS.md)

Welcome to the Opsway backend repository guidelines. This document provides technical instructions and constraints for modifying the Go monolith and database layers.

---

## 1. Directory Structure & Layers

The backend follows a domain-driven, clean architecture layout:

*   **[cmd/](file:///home/teis/opsway/backend/cmd)**: Main Cobra commands/entrypoints.
    *   [api.go](file:///home/teis/opsway/backend/cmd/api.go): REST API server initialization.
    *   [prober.go](file:///home/teis/opsway/backend/cmd/prober.go): Background prober daemon scheduler.
    *   [seed.go](file:///home/teis/opsway/backend/cmd/seed.go): Seeds development databases.
    *   [root.go](file:///home/teis/opsway/backend/cmd/root.go): Global configurations and validations.
*   **[internal/](file:///home/teis/opsway/backend/internal)**: Package-level modular domain layers.
    *   [entities](file:///home/teis/opsway/backend/internal/entities): Defines standard GORM schemas and model structs.
    *   [rest](file:///home/teis/opsway/backend/internal/rest): Implements endpoints, controllers, and routing rules using Echo.
    *   `connectors/`: Clients for external resources:
        *   [postgres/client.go](file:///home/teis/opsway/backend/internal/connectors/postgres/client.go): GORM PostgreSQL connector.
        *   [clickhouse/client.go](file:///home/teis/opsway/backend/internal/connectors/clickhouse/client.go): GORM ClickHouse connector.
        *   [redis/client.go](file:///home/teis/opsway/backend/internal/connectors/redis/client.go): Redis connector.

---

## 2. Command Reference & Local Dev

The local environment is fully dockerized and hot-reloaded using `reflex`:

*   **Start Infrastructure & Services**:
    ```bash
    docker-compose up
    ```
    This spins up PostgreSQL, Redis-Stack, ClickHouse, MinIO, and hot-reloads the `api` (port `8001`) and `prober` binaries.
*   **Stop Infrastructure**:
    ```bash
    docker-compose down
    ```
*   **Seeding the Database**:
    Use [seeds/teams_and_users.go](file:///home/teis/opsway/backend/seeds/teams_and_users.go) to add mock data. Run database seeding via standard seeds target commands.

---

## 3. Technology Stack & Coding Standards

### Separation of Concerns
*   Ensure model files under [entities/](file:///home/teis/opsway/backend/internal/entities) remain decoupled from API handler controllers in [rest/controllers](file:///home/teis/opsway/backend/internal/rest).
*   Follow the established service-repository pattern where controllers interact with high-level services, which execute repository operations.

### Schema Migration
*   GORM handles automatic migrations on startup.
*   **Rule**: When adding or updating database entities, register your struct inside GORM's `AutoMigrate` array inside [cmd/api.go](file:///home/teis/opsway/backend/cmd/api.go).
*   Verify that any essential seeding updates are reflected in [seeds/teams_and_users.go](file:///home/teis/opsway/backend/seeds/teams_and_users.go).

### Configuration Propagation
*   Configurations are loaded via Cobra and Viper.
*   **Rule**: When introducing any new environment variable or setting, add it to the global `Config` struct in [cmd/root.go](file:///home/teis/opsway/backend/cmd/root.go) using `mapstructure` tags.
*   Add default/compose configurations in both [config.yaml](file:///home/teis/opsway/backend/config.yaml) and [config.compose.yaml](file:///home/teis/opsway/backend/config.compose.yaml).

---

## 4. Verification Checklist

Before completing any backend task:
1. Ensure the code compiles properly with `go build ./...`.
2. Run database models and schema validations.
3. Validate that hot reloading completes without errors inside docker-compose.
