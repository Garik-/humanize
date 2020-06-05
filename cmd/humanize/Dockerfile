FROM alpine:latest

RUN mkdir /app
ADD bin/humanize /app/

RUN chmod a+x /app/humanize

WORKDIR /app
CMD ["./humanize"]