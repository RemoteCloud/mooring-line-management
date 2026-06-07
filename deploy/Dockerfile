# Build the single API binary. Same image runs onboard or shore (scope via env).
FROM golang:1.26 AS build
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
