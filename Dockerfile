FROM google/golang

ADD . /gopath/src/rtlong/docker-util

WORKDIR /gopath/src/rtlong/docker-util
RUN go get
RUN go install

ENTRYPOINT ["docker-util"]
