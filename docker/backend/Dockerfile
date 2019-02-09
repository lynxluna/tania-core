FROM golang:1.10-alpine as backend_builder

WORKDIR /go/src/github.com/Tanibox/tania-core
COPY . .

RUN export BUILD_PACKAGES="upx file clang build-base curl git bzr mercurial" && export OUTPUT="tania" \
      && apk --update --no-cache add ${BUILD_PACKAGES} \
      && curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
      && dep ensure \
      && CC=clang CXX=clang++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags '-s -w -linkmode external -extldflags "-static"' -o ${OUTPUT} \
      && file ${OUTPUT} && upx ${OUTPUT} \
      && apk del ${BUILD_PACKAGES} \
      && unset BUILD_PACKAGES && unset OUTPUT

FROM busybox

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificate.crt

WORKDIR /tania

COPY --from=backend_builder /go/src/github.com/Tanibox/tania-server/tania /tania/
ADD db/sqlite/ddl.sql /tania/db/sqlite/ddl.sql
ADD resources/structure.sql /tania/resources/structure.sql

RUN mkdir -p /tania/uploads/{areas,crops}

EXPOSE 8080

ENTRYPOINT ["/tania/tania"]



