############################
# STEP 1 build base
############################
FROM golang:1.24-alpine3.21 AS build-base
ENV GOTOOLCHAIN=auto
WORKDIR /build
COPY ["go.mod", "go.sum", "./"]
RUN go mod download -x

############################
# STEP 2 image base
############################
FROM alpine:3.21 AS image-base
WORKDIR /app
RUN apk add --no-cache chromium
ENTRYPOINT [ "service" ]

############################
# STEP 3 build executable
############################
FROM build-base AS builder
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /build/bin/service main.go

############################
# STEP 4 Finalize image
############################
FROM image-base
COPY --from=builder /build/bin/service /usr/bin/service