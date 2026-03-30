FROM golang:1.26.1-bookworm
LABEL org.opencontainers.image.authors="j.knedlik@gsi.de"
RUN apt-get update
COPY . /go/lustre_exporter
WORKDIR /go/lustre_exporter
RUN make build
RUN mkdir /build
RUN mv lustre_exporter /build/lustre_exporter-$(cat VERSION)
RUN mkdir /cpy
ENTRYPOINT ["cp"]
CMD ["-r","/build","/cpy/"]
