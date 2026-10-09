# One build, two images: --target orders-api (the service under test) and
# --target stubs (its downstream services).
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/orders-api ./cmd/orders-api \
 && CGO_ENABLED=0 go build -trimpath -o /out/stubs ./cmd/stubs

FROM gcr.io/distroless/static-debian12 AS orders-api
COPY --from=build /out/orders-api /orders-api
EXPOSE 8080
ENTRYPOINT ["/orders-api"]

FROM gcr.io/distroless/static-debian12 AS stubs
COPY --from=build /out/stubs /stubs
ENTRYPOINT ["/stubs"]
