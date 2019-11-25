FROM golang:1.13.4-buster as builder

WORKDIR /app
COPY . .

RUN make build

FROM scratch
COPY --from=builder /app/bin/grafana-snitch /go/bin/grafana-snitch
CMD ["/go/bin/grafana-snitch"]