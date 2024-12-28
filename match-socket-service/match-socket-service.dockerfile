FROM alpine:latest

RUN mkdir /app

COPY matchSocketApp /app

CMD [ "/app/matchSocketApp" ]