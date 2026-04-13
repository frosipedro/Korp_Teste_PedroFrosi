FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o billing ./main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/billing .

EXPOSE 8082

CMD ["./billing"]