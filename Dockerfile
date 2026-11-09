FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/outworld ./cmd/outworld

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/outworld /usr/local/bin/outworld
WORKDIR /data
USER nonroot:nonroot
EXPOSE 8787
ENV OUTWORLD_CONFIG=/config/outworld.yaml
ENTRYPOINT ["/usr/local/bin/outworld"]
CMD ["serve"]
