FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

EXPOSE 6060
EXPOSE 8080
EXPOSE 50051

RUN make build

ENV PATH="/app/bin/:$PATH"