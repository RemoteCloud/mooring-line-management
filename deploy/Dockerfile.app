# All-in-one app image: the Go server serves both the API and the built web bundle.
# (db + minio stay as their own containers.) Build-time VITE_SCOPE/VITE_VESSEL_ID select
# the web scope; runtime WEB_DIR points the server at the bundle.

FROM node:22 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
ARG VITE_SCOPE=shore
ARG VITE_VESSEL_ID=
ENV VITE_SCOPE=$VITE_SCOPE VITE_VESSEL_ID=$VITE_VESSEL_ID
RUN npm run build

FROM golang:1.26 AS api
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=api /out/server /server
COPY --from=web /web/dist /web
ENV WEB_DIR=/web
EXPOSE 8080
ENTRYPOINT ["/server"]
