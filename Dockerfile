# syntax=docker/dockerfile:1

# Production Dockerfile for goreleaser (dockers_v2).
# The voltpilot binary is pre-built by goreleaser with the frontend embedded
# via //go:embed, so no separate frontend build stage is needed here.

FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

LABEL org.opencontainers.image.source="https://github.com/tamcore/voltpilot"
LABEL org.opencontainers.image.description="voltpilot - nearest available charger of your chosen CPO"
LABEL org.opencontainers.image.licenses="MIT"

COPY ${TARGETPLATFORM}/voltpilot /voltpilot

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/voltpilot"]
