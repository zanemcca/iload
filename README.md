# iload
instanews load balancer

## Description
This container will load balance all other containers that are linked to it. It uses the tutum API to retrieve event notifications that it then uses to rewrite the nginx configurations and reload nginx

## Usage
Check tutum.yml for an example stack using the load balancer. Starting, Stopping, Scaling and Terminating will all result in a reload of the load balancer

## Environment Variables
- VIRTUAL_HOST = The server name that the service should proxy
	default: localhost
- PORT_MAP = A mapping of the form "<nginx-port>:<exposed-service-port>,<nginx-port-2>:<exposed-service-port-2>"
	default: Map all exposed ports to port 80 
- LOCATION = A custom path to be used on the listened ports 
	default: "/"
