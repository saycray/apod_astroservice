FROM golang:1.17

ENV GOPATH=/
RUN mkdir pictures

ENV APP_PORT=8000

COPY ./ ./

RUN apt-get update
RUN apt-get -y install postgresql-client

RUN go mod download
RUN go build -o astro ./cmd/main.go

CMD ["./astro"]
