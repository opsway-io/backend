############################
# STEP 1 build base
############################
FROM golang:1.17-alpine3.13 AS build-base
RUN apk add --update --no-cache git ca-certificates build-base
WORKDIR /build
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download -x

############################
# STEP 2 image base
############################
FROM alpine:3.13 as image-base
WORKDIR /app
COPY --from=build-base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT [ "monitor" ]
CMD [ "serve" ]

############################
# STEP 3 build executable
############################
FROM build-base AS builder
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /build/bin/service main.go

############################
# STEP 4 Finalize image
############################
FROM image-base
COPY --from=builder /build/bin/service /usr/bin/monitor