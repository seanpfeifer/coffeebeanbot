#---------------
#--- Builder ---
#---------------
FROM docker.io/library/golang:1.18beta2-alpine AS builder

COPY [".", "/coffeebeanbot"]
WORKDIR /coffeebeanbot
# Make sure you turn off cgo here, because the distroless/static image doesn't have glibc
RUN CGO_ENABLED=0 go build -o /cbb ./cmd/cbb

#-----------------------
#--- Resulting image ---
#-----------------------
FROM gcr.io/distroless/static:nonroot

# Mount your secret credentials file in here
VOLUME /secrets

USER nonroot

# Copy our config (NOT secrets!)
COPY cfg.toml /bot/cfg.toml
# Copy the actual built binary
COPY --from=builder /cbb /bot/

ENTRYPOINT ["/bot/cbb"]
CMD ["-cfg", "/bot/cfg.toml", "-secrets", "/secrets/discord.toml"]
