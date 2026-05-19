FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o genx .

FROM scratch
COPY --from=build /app/genx /genx
ENTRYPOINT ["/genx"]
CMD ["--help"]
