# ---- 빌드 스테이지 ----
FROM golang:1.24-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w \
      -X github.com/zcube/commit-checker/internal/version.Version=${VERSION} \
      -X github.com/zcube/commit-checker/internal/version.Commit=${COMMIT} \
      -X github.com/zcube/commit-checker/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/commit-checker .

# ---- 최종 스테이지 ----
FROM scratch

COPY --from=builder /out/commit-checker /commit-checker
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/commit-checker"]
