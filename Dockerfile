FROM golang:1.24 as builder
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/app /app
ENV HTTP_ADDR=:3000
ENV MONGO_URI=mongodb://mongo:27017
ENV MONGO_DB=clicksdb
ENV MONGO_COLL=clicks
EXPOSE 3000
ENTRYPOINT ["/app"]
