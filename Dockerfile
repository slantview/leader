FROM ubuntu:18.04

ENV LEADER_ELECTION_BACKEND "etcd"
ENV LEADER_ELECTION_URL "localhost:2379"

RUN apt-get update -y && apt-get install -y ca-certificates

ADD ./bin/linux/leader /app/leader

EXPOSE 8080 

ENTRYPOINT ["/app/leader"]