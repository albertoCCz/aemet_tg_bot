FROM golang:1.22

VOLUME /usr/src/app/templates
VOLUME /usr/src/app/pdfs-registry

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN echo "Building..."
RUN go build -v -o /usr/local/bin/app ./...

RUN echo "Running test"
RUN go test -v

ENTRYPOINT ["./aemet_tg_bot"]
CMD ["init", "--bot-config=./botConfig.json"]
