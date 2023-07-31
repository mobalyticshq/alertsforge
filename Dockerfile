# syntax=docker/dockerfile:1

ARG IMAGE
ARG TAG
FROM $IMAGE:$TAG as build
ARG BUILD_DATE
ARG BUILD_COMMIT
ARG BUILD_VERSION
ARG BINARY_NAME
ARG CI_PIPELINE_ID

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED="0"

COPY . /src
WORKDIR /src

RUN go build -o /src/alertsForge

# Deploy the application binary into a lean image
FROM ubuntu:22.04 AS build-release-stage
ENV CLOUD_SDK_VERSION=429.0.0
# Install additional dependencies
RUN apt-get update && apt-get install --no-install-recommends -qqy \
        apt-transport-https \
        bash \
        ca-certificates \
        curl \
        git \
        gpg-agent \
        jq \
        lsb-release \
        make \
        nano \
        openssh-client \
        openssl \
        python3-crcmod \
        python3-dev \
        software-properties-common \
        unzip \
        gnupg && rm -rf /var/lib/apt/lists/*
RUN export CLOUD_SDK_REPO="cloud-sdk" && \
    echo "deb https://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    apt-get update && apt-get install --no-install-recommends -y google-cloud-cli=${CLOUD_SDK_VERSION}-0 \
        google-cloud-cli-gke-gcloud-auth-plugin=${CLOUD_SDK_VERSION}-0 \
        kubectl && rm -rf /var/lib/apt/lists/*
RUN    gcloud --version && kubectl version --client


WORKDIR /

COPY --from=build /src/alertsForge /alertsForge

EXPOSE 8080

RUN useradd -ms /bin/bash app
USER app

ENTRYPOINT ["/alertsForge"]