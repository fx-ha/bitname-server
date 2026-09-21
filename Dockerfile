FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /bitname-server .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bitname-server /bitname-server
EXPOSE 6002
ENTRYPOINT ["/bitname-server"]
