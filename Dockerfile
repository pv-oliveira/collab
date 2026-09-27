# Estágio 1: compila. Imagem grande (Go completo), usada só no build.
# A versão do Go precisa acompanhar o go.mod.
FROM golang:1.26.8-alpine AS build
WORKDIR /src

# Dependências antes do código: mudar um .go não invalida o cache do download.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 gera binário estático (roda sem libc); -trimpath e -s -w
# removem caminhos locais e símbolos de debug, deixando o binário menor.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Estágio 2: só o binário. Sem shell, sem compilador, usuário não-root.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
