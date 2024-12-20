FROM alpine:latest

RUN mkdir /app

COPY matchApp /app

CMD [ "/app/matchApp" ]