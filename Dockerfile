FROM golang:1.24-bullseye AS builder

WORKDIR /app

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-w -s" -o /bin/service ./cmd/service/...

FROM gcr.io/distroless/base-debian11:nonroot
USER nonroot:nonroot
WORKDIR /
COPY --from=builder /bin/sleep /bin/sleep
COPY --from=builder /bin/service /bin/service

CMD ["/bin/service"]
