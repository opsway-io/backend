############################
# STEP 1 build base
############################
FROM golang:1.18-alpine3.16 AS build-base
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download -x

############################
# STEP 2 image base
############################
FROM alpine:3.16 as image-base
WORKDIR /app
ENTRYPOINT [ "service" ]

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
COPY --from=builder /build/bin/service /usr/bin/service