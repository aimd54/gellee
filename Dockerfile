# Copyright The gellee Authors
# SPDX-License-Identifier: Apache-2.0

FROM --platform=$BUILDPLATFORM node:26.12.1-bookworm-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist/ web/dist/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags '-s -w' -o /gellee ./cmd/gellee

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /gellee /gellee
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/gellee", "serve", "--state", "/state", "--listen", "0.0.0.0:8080"]
