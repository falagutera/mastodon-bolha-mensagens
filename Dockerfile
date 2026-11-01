# --- build -------------------------------------------------------------------
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY templates ./templates
COPY assets ./assets
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /justmytoots .

# --- runtime -----------------------------------------------------------------
# distroless static: no shell, ca-certs + a nonroot user, tiny.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /justmytoots /justmytoots
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/justmytoots", "-healthcheck"]
ENTRYPOINT ["/justmytoots"]
