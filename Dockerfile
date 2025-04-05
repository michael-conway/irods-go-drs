FROM golang:1.10 AS build
WORKDIR /go/src
COPY irods-drs ./go
COPY main.go .

ENV CGO_ENABLED=0
RUN irods-drs get -d -v ./...

RUN irods-drs build -a -installsuffix cgo -o swagger .

FROM scratch AS runtime
COPY --from=build /irods-drs/src/swagger ./
EXPOSE 8080/tcp
ENTRYPOINT ["./swagger"]
