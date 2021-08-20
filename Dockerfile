FROM golang:1.13
WORKDIR /go/src/github.com/tynany/frr_exporter
COPY . /go/src/github.com/tynany/frr_exporter
RUN make setup_promu
RUN ./promu build
RUN ls -lah

FROM alpine:3.14.1
WORKDIR /app
COPY --from=0 /go/src/github.com/tynany/frr_exporter/frr_exporter .
EXPOSE 9342
ENTRYPOINT [ "./frr_exporter"]