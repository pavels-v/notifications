FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/migrations ./cmd/migrations
RUN CGO_ENABLED=0 go build -o /out/fake-messagebird ./test/fake/messagebird/cmd

# An image of its own, so the stand-in provider cannot reach the one that ships.
FROM alpine:3.20 AS fake-messagebird
RUN adduser -D -u 10001 notifications
COPY --from=build /out/fake-messagebird /usr/local/bin/fake-messagebird
USER notifications
EXPOSE 8099
ENTRYPOINT ["/usr/local/bin/fake-messagebird"]

# Last on purpose: a build without an explicit target must produce the service, never the fake.
FROM alpine:3.20 AS service
RUN adduser -D -u 10001 notifications
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/migrations /usr/local/bin/migrations
USER notifications
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
