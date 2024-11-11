FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY gatewayApp /app

CMD [ "/app/gatewayApp" ]