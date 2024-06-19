FROM moj-sgp-armory.platform.internal/dockerhub/golang:1.18-alpine

RUN apk update && apk upgrade && apk add --no-cache git procps && apk add --no-cache --update go gcc g++
RUN go env -w GOPRIVATE=github.com/ShareChat

WORKDIR /lte-cohort-service

ARG GITHUB_TOKEN
ARG DEPLOYMENT_ID

RUN git config \
    --global \
    url."https://${GITHUB_TOKEN}@github.com".insteadOf \
    "https://github.com"

# These 3 steps helps to speedup builds when using docker cache
# Copying module files for building image
COPY go.mod .
COPY go.sum .

# Download modules to local cache
RUN go mod download

# Order of these 2 steps is important
# If go mod tidy is ran before, it removes all
# dependend modules as no source files are present at this point.
COPY . .
RUN go mod tidy

RUN go build -o sc-live-topic-table-database-cleanup-producer

ENTRYPOINT ["./sc-live-topic-table-database-cleanup-producer"]