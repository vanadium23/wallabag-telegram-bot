FROM golang:1.23 AS build-env
WORKDIR /src

RUN apt install git gcc

ADD . /src

RUN go mod download && go mod verify
RUN go build -o wallabot ./cmd/cli

# final stage
FROM golang:1.23
WORKDIR /root/.config/t.me
COPY --from=build-env /src/wallabot /app/
ENTRYPOINT /app/wallabot
