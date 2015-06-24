# iload
instanews load balancer

## Description
This container will load balance all other containers that are linked to it. It uses the tutum API to retrieve event notifications that it then uses to rewrite the nginx configurations and reload nginx

## Usage
Check tutum.yml for an example stack using the load balancer. Starting, Stopping, Scaling and Terminating will all result in a reload of the load balancer
