# Optimized for fast local builds.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM:-.}/intervals-deduper-HE /app/intervals-deduper-HE

# Running as rootless distroless
USER nonroot:nonroot
ENTRYPOINT ["/app/intervals-deduper-HE"]
