FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/yanakilla ./cmd/yanakilla
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/indexer ./cmd/indexer

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/yanakilla /usr/local/bin/yanakilla
COPY --from=build /out/indexer /usr/local/bin/indexer
USER app
EXPOSE 3000
ENTRYPOINT ["yanakilla"]
CMD ["-port", "3000"]
