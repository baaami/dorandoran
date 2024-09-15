FROM alpine:latest

RUN mkdir /app

COPY consumerApp /app

CMD [ "/app/consumerApp" ]