# Whenever the Go version is updated here, .circle/config.yml and .promu.yml should also be updated.
FROM golang:1.17
WORKDIR /go/src/github.com/tynany/frr_exporter
COPY . /go/src/github.com/tynany/frr_exporter
RUN make setup_promu
RUN ./promu build
RUN ls -lah

FROM frrouting/frr:v7.5.1
WORKDIR /app
COPY --from=0 /go/src/github.com/tynany/frr_exporter/frr_exporter .
EXPOSE 9342
ENTRYPOINT [ "./frr_exporter"]