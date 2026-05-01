# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/gitgram ./cmd/gitgram

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/gitgram /gitgram

EXPOSE 8080

ENTRYPOINT ["/gitgram"]
