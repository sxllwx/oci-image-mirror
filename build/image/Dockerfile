FROM alpine:latest

ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app

ARG app

RUN apk add -U tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
      echo "Asia/Shanghai" > /etc/timezone

COPY $TARGETPLATFORM/$app /app/$app

ENV APP_NAME=$app

ENTRYPOINT ./${APP_NAME}