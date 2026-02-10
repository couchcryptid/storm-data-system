FROM golang:1.24-alpine AS build

RUN apk add --no-cache busybox-static

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /mock-server .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /bin/busybox.static /bin/busybox
COPY --from=build /mock-server /mock-server
COPY data/ /data/

EXPOSE 8080

ENTRYPOINT ["/mock-server"]
