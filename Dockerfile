FROM golang:1.11.0

RUN mkdir /content-service
RUN apt-get update && apt-get -y install redis-tools
WORKDIR /content-service
COPY . /content-service/

EXPOSE 8000

CMD ["/bin/bash","-c","./content-service-run.sh"]
