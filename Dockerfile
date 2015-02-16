FROM golang

ADD . /weatherstation
WORKDIR /weatherstation

RUN make clean && make build
