FROM golang:1.18-buster AS build-env
WORKDIR /src

RUN apt install git gcc

ADD . /src

RUN go mod download && go mod verify
RUN go build -o wallabag-bot

# final stage
FROM golang:1.18-buster
WORKDIR /root/.config/t.me
COPY --from=build-env /src/wallabag-bot /app/
ENTRYPOINT /app/wallabag-bot
