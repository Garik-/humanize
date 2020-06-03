FROM alpine:latest

RUN mkdir /app
ADD bin/scan /app/

RUN chmod a+x /app/scan

WORKDIR /app
CMD ["./scan"]