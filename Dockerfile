FROM nginx 

MAINTAINER Zane McCaig

# Update the repositories
RUN apt-get update

# Install go , mercurial and git
RUN apt-get install -y golang mercurial git

# Copy all of our source files
COPY . /src
WORKDIR /src

ENV GOPATH /src

RUN cd /src && \
  go get github.com/tutumcloud/go-tutum/tutum && \
  go build -o iload && \
  rm -r /etc/nginx && \
  ln -s /src/nginx /etc/nginx && \
  echo "expose_php = Off" >> /etc/php.ini

EXPOSE 80

CMD ./iload  
