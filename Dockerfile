FROM nginx 

MAINTAINER Zane McCaig

# Update the repositories
RUN apt-get update

# Install go , mercurial and git
RUN apt-get install -y mercurial git wget && \
  wget https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz && \
  tar -C /usr/local -xvf go1.4.2.linux-amd64.tar.gz

# Copy all of our source files
COPY . /src
WORKDIR /src

ENV GOPATH /src
ENV PATH $PATH:/usr/local/go/bin

RUN cd /src && \
  go get github.com/tutumcloud/go-tutum/tutum && \
  go build -o iload && \
  rm -r /etc/nginx && \
  ln -s /src/nginx /etc/nginx

EXPOSE 80

RUN go version

CMD ./iload  
