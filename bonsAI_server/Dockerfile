FROM node:22-bookworm-slim AS frontend-builder
WORKDIR /app/bonsAI_front

COPY bonsAI_front/package.json ./
RUN npm install

COPY bonsAI_front/index.html ./index.html
COPY bonsAI_front/vite.config.js ./vite.config.js
COPY bonsAI_front/public ./public
COPY bonsAI_front/src ./src
RUN npm run build

FROM golang:1.25-bookworm AS backend-builder
WORKDIR /app/bonsAI_server

COPY bonsAI_server/go.mod bonsAI_server/go.sum ./
RUN go mod download

COPY bonsAI_server ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/bonsai_server .

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=backend-builder /out/bonsai_server /app/bonsai_server
COPY --from=frontend-builder /app/bonsAI_front/dist /app/static

ENV BONSAI_SERVER_ADDR=:8080
ENV BONSAI_STATIC_DIR=/app/static

EXPOSE 8080

CMD ["/app/bonsai_server"]
