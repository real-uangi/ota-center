FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod ./
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/ota-center .

FROM gcr.io/distroless/static-debian12

WORKDIR /app

ENV PORT=8080
ENV OTA_DATA_DIR=/data

COPY --from=builder /out/ota-center /usr/local/bin/ota-center

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/ota-center"]
