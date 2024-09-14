FROM alpine:latest

RUN mkdir /app

COPY gatewayApp /app

CMD [ "/app/gatewayApp" ]