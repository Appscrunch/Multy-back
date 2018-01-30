FROM golang:1.9.2

RUN mkdir -p go/src/github.com/Appscrunch/Multy-back-testing
WORKDIR /go/src/github.com/Appscrunch/Multy-back-testing

COPY . /go/src/github.com/Appscrunch/Multy-back-testing

RUN go get github.com/satori/go.uuid 

CMD go run main.go --addr=localhost:5578