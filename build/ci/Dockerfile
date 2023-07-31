ARG IMAGE
ARG TAG
FROM $IMAGE:$TAG
LABEL org.label-schema.schema-version="1.0.0-rc.1" \
      org.label-schema.vendor="Mobalytics" \
      org.label-schema.name="CI alertsforge" \
      org.label-schema.description="Golang Docker image used by CI for alertsforge repo." \
      org.label-schema.vcs-url="https://github.com/mobalyticshq/alertsforge"

RUN apk add build-base
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
RUN go install github.com/boumenot/gocover-cobertura@latest
RUN go install github.com/jstemmer/go-junit-report/v2@latest
ENV GO111MODULE=on
# Install additional dependencies
RUN mkdir -p /src
COPY go.mod go.sum /src/
RUN cd /src && go mod download

