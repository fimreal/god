# go build
FROM golang:1.21 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o god .

# Portable images, People all like small ones.
FROM scratch
COPY --from=builder /src/god /god

ENTRYPOINT ["/god"]