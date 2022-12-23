# opsway backend

Mono repo for opsway backend related projects.

## Development

Everything is dockerized, so you can start development with just a few commands.

All golang code is hot-reloaded, so you don't need to restart the server after every change.

All migrations are run automatically on server start.

### Requirements

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Start

```bash
docker-compose up
```

### Stop

```bash
docker-compose down
```