FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY authApp /app

CMD [ "/app/authApp" ]