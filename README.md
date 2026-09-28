# collab

API em Go para documentos colaborativos: autenticação JWT, CRUD de documentos por usuário e edição em tempo real via WebSocket.

> 🚧 Em desenvolvimento: o hub WebSocket (`internal/ws`) está implementado e sendo integrado às rotas.

## Stack
Go · Gin · PostgreSQL (`lib/pq`) · JWT (`golang-jwt`) · WebSocket (`gorilla/websocket`)

## Arquitetura

```
cmd/api/            → entrypoint: config, injeção de dependências e rotas
migrations/         → schema do banco versionado (golang-migrate)
internal/
  config/           → configuração via variáveis de ambiente
  db/               → conexão com PostgreSQL
  handlers/         → camada HTTP (auth, documentos)
  services/         → regras de negócio
  repositories/     → acesso a dados (SQL)
  middleware/       → autenticação JWT
  models/           → entidades
  ws/               → hub WebSocket: um canal de broadcast por documento
```

O fluxo segue `handler → service → repository`: o handler só trata HTTP, o service concentra a regra de negócio e o repository o SQL. Isso mantém cada camada testável isoladamente.

**Tempo real:** cada documento tem seu próprio conjunto de clientes no `Hub`. Registro, saída e broadcast passam por canais, e cada cliente tem goroutines separadas de leitura e escrita (`ReadPump`/`WritePump`).

## Endpoints

| Método | Rota | Auth |
|---|---|---|
| POST | `/auth/register` | — |
| POST | `/auth/login` | — |
| POST | `/documents` | JWT |
| GET | `/documents` | JWT |
| GET | `/documents/:id` | JWT |
| PUT | `/documents/:id` | JWT |

## Banco de dados

O schema fica versionado em `migrations/` e é aplicado com o [golang-migrate](https://github.com/golang-migrate/migrate):

```sh
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/collab?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up        # aplica as pendentes
migrate -path migrations -database "$DATABASE_URL" down 1    # desfaz a última
migrate create -ext sql -dir migrations -seq nome_da_mudanca # cria uma nova
```

O CI roda `up → down → up` num Postgres real a cada PR.

## Como rodar

### Com Docker (recomendado)

```sh
docker compose up --build
```

Sobe Postgres, Redis, aplica as migrations e inicia **duas réplicas** da API:

| Serviço | Endereço |
|---|---|
| API réplica 1 | `http://localhost:8080` |
| API réplica 2 | `http://localhost:8081` |
| Postgres | `localhost:5433` (usuário e senha `postgres`, banco `collab`) |

As duas réplicas compartilham Postgres e Redis: uma edição enviada por WebSocket à réplica 1 chega aos clientes conectados na réplica 2.
`JWT_SECRET` e `ALLOWED_ORIGINS` podem ser definidos no `.env`; sem eles, valem valores de desenvolvimento.

### Imagem publicada

Cada merge na `main` publica a imagem no GitHub Container Registry, com as tags `latest` e `sha-<commit>` (imutável, para deploy e rollback):

```sh
docker pull ghcr.io/pv-oliveira/collab:latest
docker run -p 8080:8080 \
  -e DATABASE_URL=... -e REDIS_URL=... -e JWT_SECRET=... -e ALLOWED_ORIGINS=... \
  ghcr.io/pv-oliveira/collab:latest
```

### Sem Docker

```sh
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/collab?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="troque-isto"
export ALLOWED_ORIGINS="http://localhost:5173"   # origens de navegador aceitas (CORS e WebSocket)
go run ./cmd/api
```

A API sobe em `http://localhost:8080`.

## Testes

```sh
go test ./...                                    # unitários; os de integração são pulados
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/collab_test?sslmode=disable" go test ./...
```

Os testes de integração usam um banco separado (`collab_test`) com as migrations aplicadas. No CI eles rodam contra um Postgres 17 real.

## Próximos passos
- [x] Expor a rota WebSocket `/documents/:id/ws`
- [x] Migrations do banco
- [x] Testes do hub e dos services
- [x] Dockerfile + docker-compose
- [x] CI com GitHub Actions
