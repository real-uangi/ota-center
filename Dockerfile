FROM gcr.io/distroless/static-debian12

WORKDIR /app

ENV PORT=8080
ENV OTA_DATA_DIR=/data

COPY ota-center /usr/local/bin/ota-center

EXPOSE 8765
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/ota-center"]
