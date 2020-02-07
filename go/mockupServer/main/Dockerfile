FROM golang:alpine AS build-env
ADD main.go /src/
WORKDIR /src
RUN apk add git --no-cache && go get -d -v . && go build -o app
# DOWNLOAD DEPS
# ADD go.mod go.sum /src/
# WORKDIR /src
# RUN go mod download
# # BUILD BIN
# ADD main.go utils.go /src/
# RUN go build -o app
# # COPY BIN INTO CLEAN IMAGE
FROM alpine
COPY --from=build-env /src/app /
ENTRYPOINT ./app $OPTS