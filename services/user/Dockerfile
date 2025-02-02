FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app

COPY userApp /app

CMD [ "/app/userApp" ]