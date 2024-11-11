FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY consumerApp /app

CMD [ "/app/consumerApp" ]