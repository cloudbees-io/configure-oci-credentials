FROM golang:1.20.5-alpine3.18 AS build

WORKDIR /work

COPY go.mod* go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /build-out/ .

FROM scratch

COPY --from=build /build-out/* /usr/bin/

WORKDIR /cloudbees/home

ENV HOME=/cloudbees/home
ENV PATH=/usr/bin

ENTRYPOINT ["configure-oci-credentials"]
