FROM alpine:latest

RUN apk --no-cache add tzdata

RUN mkdir /app
RUN mkdir /app/images

COPY gatewayApp /app
COPY ./resource/img/* /app/images

CMD [ "/app/gatewayApp" ]