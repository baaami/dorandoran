FROM alpine:latest

RUN mkdir /app

COPY realtimeApp /app

CMD [ "/app/realtimeApp" ]