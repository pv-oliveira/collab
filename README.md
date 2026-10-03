# collab

[![CI](https://github.com/pv-oliveira/collab/actions/workflows/ci.yml/badge.svg)](https://github.com/pv-oliveira/collab/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/pv-oliveira/collab)](go.mod)
[![Imagem](https://img.shields.io/badge/ghcr.io-pv--oliveira%2Fcollab-2496ED?logo=docker&logoColor=white)](https://github.com/pv-oliveira/collab/pkgs/container/collab)
[![Licença: MIT](https://img.shields.io/badge/licen%C3%A7a-MIT-green)](LICENSE)

API em Go para **edição colaborativa de documentos em tempo real**, feita para rodar com **várias réplicas**: uma edição recebida por uma instância chega aos clientes conectados em qualquer outra, via **Redis Pub/Sub**.

- **Tempo real distribuído:** WebSocket + Redis Pub/Sub entre réplicas, sem eco e sem duplicação.
- **Segurança:** JWT (só HS256) revogável no logout, autorização por dono do documento, rate limit no login e no cadastro (compartilhado entre réplicas), CORS e `CheckOrigin` por lista de origens, hash da senha nunca exposto.
- **Qualidade:** 87,5% de cobertura, testes com o detector de corrida (`-race`), testes de integração com Postgres real e testes de mutação.
- **Entrega:** Docker (imagem de ~10 MB, sem shell, não-root), `docker compose` com duas réplicas, CI com 5 checks obrigatórios e CD publicando no GitHub Container Registry.

## Demo em 1 minuto

```sh
docker compose up --build
```

1. Abra **http://localhost:5173**, crie uma conta na réplica `:8080` e crie um documento.
2. Clique em **"Abrir na outra réplica"**: uma nova aba abre o mesmo documento conectado à `:8081`.
3. Digite numa aba e veja o texto aparecer na outra — a mensagem atravessou de uma instância da API para a outra pelo Redis.

## Arquitetura

```mermaid
flowchart LR
    subgraph Navegador
        A["Aba 1"]
        B["Aba 2"]
    end
    A -->|WebSocket| API1["api-1 :8080"]
    B -->|WebSocket| API2["api-2 :8081"]
    API1 <-->|Pub/Sub collab:events| R[("Redis")]
    API2 <-->|Pub/Sub collab:events| R
    API1 --> PG[("Postgres")]
    API2 --> PG
    M["migrate"] -.->|schema versionado| PG
```

O que acontece quando alguém digita:

```mermaid
sequenceDiagram
    participant A as Aba 1
    participant I1 as api-1
    participant R as Redis
    participant I2 as api-2
    participant B as Aba 2
    A->>I1: mensagem (WebSocket)
    I1->>R: PUBLISH collab:events {documento, sender_id, conteúdo}
    R-->>I1: evento
    R-->>I2: evento
    Note over I1: entrega às outras conexões locais<br/>e pula a de origem (sender_id)
    I2->>B: conteúdo (WebSocket)
```

A instância **só publica no Redis** e todas — inclusive a de origem — entregam a partir da inscrição. Com um único caminho, a mensagem nunca chega duplicada. Dentro de cada instância, só a goroutine do `Hub` acessa o mapa de conexões; as demais conversam com ela por canais, sem corrida de dados.

### Código

```
cmd/api/            → entrypoint: configuração, injeção de dependências e rotas
internal/
  config/           → variáveis de ambiente
  db/               → conexão com Postgres (falha cedo se o banco estiver fora)
  handlers/         → camada HTTP: converte erros de domínio em status
  services/         → regras de negócio
  repositories/     → SQL; traduz erros do Postgres em erros de domínio
  middleware/       → JWT, CORS e regra de origens
  models/           → entidades e erros de domínio
  ws/               → hub WebSocket distribuído via Redis
  testutil/         → helpers dos testes de integração
migrations/         → schema versionado (golang-migrate)
demo/               → página de demonstração (HTML/JS sem dependências)
```

O fluxo é `handler → service → repository`. Erros de domínio (como `ErrEmailTaken`) ficam em `models`, então a camada HTTP não conhece o Postgres.

## Stack

Go 1.26 · Gin · PostgreSQL 17 · Redis 8 · gorilla/websocket · golang-jwt · golang-migrate · Docker · GitHub Actions

## API

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| POST | `/auth/register` | — | Cria conta (`409` se o e-mail já existe) |
| POST | `/auth/login` | — | Retorna o JWT (válido por 24h) |
| POST | `/auth/logout` | Bearer | Revoga o token usado (`204`); ele passa a receber `401` no HTTP e no WebSocket |

`/auth/register` e `/auth/login` aceitam **10 requisições por minuto por IP** (cada rota com seu contador). Acima disso: `429` com `Retry-After` em segundos.
| POST | `/documents` | Bearer | Cria documento |
| GET | `/documents` | Bearer | Lista os documentos do usuário |
| GET | `/documents/:id` | Bearer | Lê um documento (`404` se não for do usuário) |
| PUT | `/documents/:id` | Bearer | Salva título e conteúdo |
| GET | `/documents/:id/ws` | subprotocolo | Canal de edição em tempo real |

**WebSocket no navegador:** navegadores não permitem headers customizados no WebSocket, então o token vai no subprotocolo:

```js
const ws = new WebSocket(`ws://localhost:8080/documents/${id}/ws`, ["access_token", token]);
ws.onmessage = (e) => console.log(e.data);
ws.send("novo conteúdo");
```

## Como rodar

### Docker (recomendado)

```sh
docker compose up --build
```

| Serviço | Endereço |
|---|---|
| Demo | http://localhost:5173 |
| API réplica 1 | http://localhost:8080 |
| API réplica 2 | http://localhost:8081 |
| Postgres | `localhost:5433` (usuário e senha `postgres`, banco `collab`) |

A ordem é garantida por healthchecks: Postgres e Redis saudáveis → `migrate` aplica o schema e termina → as APIs sobem.

### Imagem publicada

Cada merge na `main` publica a imagem com as tags `latest` e `sha-<commit>` (imutável, para deploy e rollback):

```sh
docker run -p 8080:8080 \
  -e DATABASE_URL=... -e REDIS_URL=... -e JWT_SECRET=... -e ALLOWED_ORIGINS=... \
  ghcr.io/pv-oliveira/collab:latest
```

### Sem Docker

Requer Go 1.26, Postgres e Redis locais.

```sh
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/api
```

### Configuração

| Variável | Obrigatória | Exemplo |
|---|---|---|
| `DATABASE_URL` | sim | `postgres://postgres:postgres@localhost:5432/collab?sslmode=disable` |
| `REDIS_URL` | sim | `redis://localhost:6379` |
| `JWT_SECRET` | sim | segredo longo e aleatório |
| `ALLOWED_ORIGINS` | não | `http://localhost:5173` — origens de navegador aceitas (CORS e WebSocket), separadas por vírgula. Vazia bloqueia todo navegador. |

## Migrations

```sh
migrate -path migrations -database "$DATABASE_URL" up        # aplica as pendentes
migrate -path migrations -database "$DATABASE_URL" down 1    # desfaz a última
migrate create -ext sql -dir migrations -seq nome_da_mudanca # cria uma nova
```

## Testes

```sh
go test ./...        # unitários; os de integração são pulados
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/collab_test?sslmode=disable" go test ./...
```

- **Unitários:** JWT e revogação, CORS, origens, rate limit (Redis em memória), serialização, configuração.
- **Integração** (Postgres real): repositories, services, handlers HTTP e um **WebSocket ponta a ponta** com conexões reais via `httptest.Server`.
- **Hub distribuído:** duas instâncias ligadas ao mesmo Redis em memória (`miniredis`).
- **Testes de mutação:** cada bug corrigido foi reintroduzido de propósito para confirmar que algum teste falha.

## CI/CD

Todo PR precisa passar em 5 checks antes do merge:

| Check | O que garante |
|---|---|
| `test` | build, `go vet` e `go test -race` com Postgres real |
| `lint` | `golangci-lint` |
| `govulncheck` | nenhuma vulnerabilidade **alcançável** pelo código (também roda toda segunda) |
| `migrations` | `up → down → up` num Postgres real |
| `docker` | a imagem compila, o compose é válido e o Go do `Dockerfile` é o mesmo do `go.mod` |

Em cada push na `main`, o job `publish` roda só depois dos 5 checks e publica a imagem no GHCR.

## Decisões técnicas

| Decisão | Motivo | PR |
|---|---|---|
| Publicar eventos **só** no Redis | Um único caminho: sem mensagens duplicadas na instância de origem | [#1](https://github.com/pv-oliveira/collab/pull/1) |
| JWT no subprotocolo do WebSocket | Navegadores não enviam headers customizados; não vaza em logs de URL | [#1](https://github.com/pv-oliveira/collab/pull/1) |
| `govulncheck` além do Dependabot | Mede alcançabilidade e cobre a biblioteca padrão | [#14](https://github.com/pv-oliveira/collab/pull/14) |
| Migrations em container próprio | Sem corrida entre réplicas na subida | [#15](https://github.com/pv-oliveira/collab/pull/15) |
| Erros de domínio em `models` | A camada HTTP não depende do Postgres | [#16](https://github.com/pv-oliveira/collab/pull/16) |
| Testes de integração com Postgres real | Testam o SQL de verdade, sem refatorar para interfaces | [#18](https://github.com/pv-oliveira/collab/pull/18) |
| Imagem distroless `nonroot` | ~10 MB, sem shell, mínima superfície de ataque | [#20](https://github.com/pv-oliveira/collab/pull/20) |
| `publish` com `needs` nos 5 checks | O próprio GitHub garante que só publica com CI verde | [#21](https://github.com/pv-oliveira/collab/pull/21) |
| CORS próprio com a mesma lista do WebSocket | Uma única regra de origens; sem dependência nova | [#24](https://github.com/pv-oliveira/collab/pull/24) |
| Rate limit no Redis, janela fixa em `MULTI` | Vale entre réplicas; `SET NX EX` + `INCR` atômicos evitam chave sem TTL; Redis fora → deixa passar | [#28](https://github.com/pv-oliveira/collab/issues/28) |
| `SetTrustedProxies(nil)` | Sem proxy na frente: um `X-Forwarded-For` falso não troca o IP do rate limit | [#28](https://github.com/pv-oliveira/collab/issues/28) |
| Logout por denylist do `jti` no Redis | TTL = tempo restante do token: a chave some sozinha; o JWT continua stateless | [#29](https://github.com/pv-oliveira/collab/issues/29) |
| Verificação única (`middleware.Tokens`) para HTTP e WebSocket | Nenhum caminho esquece a checagem de revogação | [#29](https://github.com/pv-oliveira/collab/issues/29) |
| Revogação *fail closed* (`503`), rate limit *fail open* | Sem o Redis, um token revogado voltaria a valer; já o rate limit é só defesa extra | [#29](https://github.com/pv-oliveira/collab/issues/29) |

O histórico completo — contexto, alternativas e critérios de aceite — está nas [issues](https://github.com/pv-oliveira/collab/issues?q=is%3Aissue) e nos [PRs](https://github.com/pv-oliveira/collab/pulls?q=is%3Apr).

## Limitações conhecidas

- **A última edição vence.** Não há algoritmo de mesclagem (CRDT ou Operational Transformation): edições simultâneas no mesmo documento se sobrescrevem.
- **O tempo real não salva sozinho.** O WebSocket transmite; a gravação é feita pelo `PUT /documents/:id`.
- **Tokens não são revogáveis.** Um token continua válido por até 24h, mesmo se o usuário for apagado.
- **Sem limite de requisições** (*rate limiting*) nas rotas de autenticação.

## Licença

[MIT](LICENSE) © 2026 Paulo Victor Oliveira
