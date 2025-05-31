FROM debian:bookworm-slim
RUN apt update && apt install -y libsqlite3-0
WORKDIR /app
COPY static ./static
COPY templates ./templates
COPY linuxBuild .
CMD ["./linuxBuild"]