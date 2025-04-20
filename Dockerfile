# argument for Go version
ARG GO_VERSION=1.24
 
# STAGE 1: building the executable
FROM golang:${GO_VERSION}-alpine AS build
 
RUN apk add --no-cache git
WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
# Build the executable
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -buildid=" -trimpath \
    -o /app ./cmd/gw2-alliance-bot/main.go
 
# STAGE 2: build the container to run
FROM gcr.io/distroless/static AS final
 
USER nonroot:nonroot
 
# copy compiled app
COPY --from=build --chown=nonroot:nonroot /app /app
 
# run binary; use vector form
ENTRYPOINT ["/app"]