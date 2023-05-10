# Build layer
FROM golang:1.19-alpine AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

# Run layer
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /app/app .
USER 65532:65532
ENTRYPOINT ["/app/app"]