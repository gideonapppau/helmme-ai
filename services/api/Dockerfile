FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /api .

FROM alpine:3.20
COPY --from=build /api /api
CMD ["/api"]
