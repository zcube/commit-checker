# syntax=docker/dockerfile:1

# ---- 빌드 스테이지 ----
FROM golang:1.26-alpine AS builder

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
# commit-checker는 diff/show/ls-files 등을 위해 git CLI를 호출하므로
# scratch 이미지 대신 최소 Alpine 런타임을 사용합니다.
FROM alpine:3.23

RUN apk add --no-cache ca-certificates git git-lfs \
    && git config --system --add safe.directory /repo

COPY --from=builder /out/commit-checker /usr/local/bin/commit-checker

# --user UID:GID로 실행해도 전역 설정/캐시 탐색이 root 홈에서 실패하지 않게 합니다.
ENV HOME=/tmp
WORKDIR /repo

ENTRYPOINT ["commit-checker"]
