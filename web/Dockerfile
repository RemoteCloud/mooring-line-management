# Build the PWA and serve it via nginx. Build-time VITE_SCOPE selects onboard/shore.
FROM node:22 AS build
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
ARG VITE_SCOPE=shore
ARG VITE_VESSEL_ID=
ENV VITE_SCOPE=$VITE_SCOPE VITE_VESSEL_ID=$VITE_VESSEL_ID
RUN npm run build

FROM nginx:alpine
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
