FROM gcr.io/distroless/static-debian12

ARG TARGETPLATFORM

WORKDIR /app

ENV PORT=8080
ENV OTA_DATA_DIR=/data

COPY ${TARGETPLATFORM}/ota-center /usr/local/bin/ota-center

EXPOSE 8765
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/ota-center"]
