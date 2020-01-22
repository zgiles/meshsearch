FROM golang:latest as builder
WORKDIR /go/src/github.com/zgiles/meshsearch
ADD . /go/src/github.com/zgiles/meshsearch/
RUN make all

FROM alpine:latest
WORKDIR /
COPY --from=builder /go/src/github.com/zgiles/meshsearch/static /static
COPY --from=builder /go/src/github.com/zgiles/meshsearch/config.json .
COPY --from=builder /go/src/github.com/zgiles/meshsearch/meshsearch .
CMD ["/meshsearch"]
