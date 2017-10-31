FROM ubuntu:17.04

MAINTAINER Daniil Kotelnikov

RUN apt-get -y update && apt-get install -y wget git


ENV PGVER 9.6
RUN apt-get install -y postgresql-$PGVER

USER postgres

RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER docker WITH SUPERUSER PASSWORD 'docker';" &&\
    createdb -O docker forum_db &&\
    /etc/init.d/postgresql stop


RUN echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/$PGVER/main/pg_hba.conf

RUN echo "listen_addresses='*'" >> /etc/postgresql/$PGVER/main/postgresql.conf

#RUN echo "logging_collector=on" >> /etc/postgresql/9.5/main/postgresql.conf
#RUN echo "log_statement='ddl'" >> /etc/postgresql/9.5/main/postgresql.conf
#RUN echo "log_directory='/var/log/postgresql'" >> /etc/postgresql/9.5/main/postgresql.conf

EXPOSE 5432

USER root

#
# Сборка проекта
#

# Установка golang
RUN wget https://storage.googleapis.com/golang/go1.9.1.linux-amd64.tar.gz

RUN tar -C /usr/local -xzf go1.9.1.linux-amd64.tar.gz && \
    mkdir go && mkdir go/src && mkdir go/bin && mkdir go/pkg

# Выставляем переменную окружения для сборки проекта
ENV GOPATH $HOME/go

ENV PG_PORT 5432

ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH


RUN echo "synchronous_commit='off'" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "max_wal_size = 1GB" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "shared_buffers = 512MB" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "effective_cache_size = 512MB" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "work_mem = 256MB" >> /etc/postgresql/$PGVER/main/postgresql.conf
#RUN echo "log_duration = on" >> /etc/postgresql/$PGVER/main/postgresql.conf
#RUN echo "log_min_duration_statement = 20" >> /etc/postgresql/$PGVER/main/postgresql.conf
#RUN echo "max_prepared_transactions = 8" >> /etc/postgresql/$PGVER/main/postgresql.conf

RUN go get -u github.com/mailru/easyjson/...

ADD ./ $GOPATH/src/github.com/zwirec/tech-db/

WORKDIR $GOPATH/src/github.com/zwirec/tech-db/

RUN go generate ./models

RUN go install github.com/zwirec/tech-db/

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql/data"]

EXPOSE 5000
EXPOSE 1111

CMD service postgresql start && tech-db