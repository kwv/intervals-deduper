# Optimized for fast local builds.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM:-.}/intervals-deduper /app/intervals-deduper

# Running as rootless distroless
USER nonroot:nonroot
ENTRYPOINT ["/app/intervals-deduper"]
