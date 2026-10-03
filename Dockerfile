FROM golang:1.23-bookworm AS build

RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

ENV CGO_ENABLED=1
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/timetable-editor ./cmd/server

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /out/timetable-editor ./timetable-editor

ENV OPEN_BROWSER=false
CMD ["./timetable-editor"]
