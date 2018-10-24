FROM golang:1.11.0

RUN mkdir /content-service
WORKDIR /content-service
COPY * /content-service

EXPOSE 8000

CMD ["/bin/bash","./content-server-run.sh"]
