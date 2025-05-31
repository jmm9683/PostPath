FROM debian:bookworm-slim
RUN apt update && apt install -y libsqlite3-0
WORKDIR /app
RUN mkdir -p local
COPY static ./static
COPY templates ./templates
COPY linuxBuild .
RUN chmod 777 local
CMD ["./linuxBuild"]