FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY pushApp /app

CMD [ "/app/pushApp" ]