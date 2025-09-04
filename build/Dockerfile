# BUILD =============================================================================================
ARG GOLANG_VERSION=1.25
FROM golang:${GOLANG_VERSION}-alpine AS builder

WORKDIR /build
COPY . .

RUN go get ./... \
    && CGO_ENABLED=0 go test ./... -v \
    && CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o s3-nuke .

# RUNTIME ===========================================================================================
FROM scratch
COPY --from=builder /build/s3-nuke /bin/
ENTRYPOINT [ "/bin/s3-nuke" ]
