FROM golang:1.11-alpine as build
RUN apk update && apk add git
WORKDIR /go/src/github.com/lucj/genx
COPY *.go ./
RUN go get github.com/lucj/genx
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o genx .

FROM scratch
COPY --from=build /go/src/github.com/lucj/genx/genx /
ENTRYPOINT ["/genx"]
CMD ["--help"]
