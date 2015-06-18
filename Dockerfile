
USE ubuntu:14.04

RUN apt-get install golang mercurial

CMD cd /src && go run main.go
