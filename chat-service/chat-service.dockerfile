FROM alpine:latest

RUN mkdir /app

COPY chatApp /app

CMD [ "/app/chatApp" ]