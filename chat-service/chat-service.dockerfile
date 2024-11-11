FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY chatApp /app

CMD [ "/app/chatApp" ]