FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service .

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=build /out/auth-service /app/auth-service

USER app
EXPOSE 8080

ENTRYPOINT ["/app/auth-service"]
