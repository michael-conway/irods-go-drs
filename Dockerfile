FROM golang:1.25-alpine AS build

WORKDIR /src

ENV CGO_ENABLED=0
ENV GOWORK=off

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY api ./api
COPY drs-support ./drs-support
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/irods-go-drs .

FROM alpine:3.22 AS runtime

RUN addgroup -S app \
    && adduser -S -G app app \
    && apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/irods-go-drs /app/irods-go-drs

EXPOSE 8080/tcp

USER app

ENTRYPOINT ["/app/irods-go-drs"]
