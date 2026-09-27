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

```sh
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/collab?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="troque-isto"
go run ./cmd/api
```

A API sobe em `http://localhost:8080`.

## Próximos passos
- [x] Expor a rota WebSocket `/documents/:id/ws`
- [x] Migrations do banco
- [ ] Testes do hub e dos services
- [ ] Dockerfile + docker-compose
- [x] CI com GitHub Actions
