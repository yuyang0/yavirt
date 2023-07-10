FROM golang:bookworm AS BUILD

# make binary
# RUN git clone https://github.com/projecteru2/yavirt.git /go/src/github.com/projecteru2/yavirt
COPY . /go/src/github.com/projecteru2/yavirt
WORKDIR /go/src/github.com/projecteru2/yavirt
ARG KEEP_SYMBOL
RUN apt update && \
    apt install -y build-essential libvirt-dev make libguestfs-dev
RUN make deps 
RUN make && ./bin/yavirtd --version

FROM debian:bookworm

RUN mkdir /etc/yavirt/ && \
    apt update && \
    apt install -y libvirt-dev libguestfs-dev
LABEL ERU=1
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtd /usr/bin/yavirtd
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtctl /usr/bin/yavirtctl
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/disk.xml /etc/yavirt/disk.xml
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/guest.xml /etc/yavirt/guest.xml
