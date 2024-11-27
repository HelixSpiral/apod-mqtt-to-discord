FROM golang:alpine AS build

# We need ca-certificates for ssl cert verification
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY main.go .

COPY go.* .

RUN go build -a -tags netgo -ldflags '-w' -v -o main .

FROM scratch

WORKDIR /app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/main .

CMD [ "/app/main" ]