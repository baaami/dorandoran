FROM alpine:latest

RUN mkdir /app

COPY chatSocketApp /app

CMD [ "/app/chatSocketApp" ]