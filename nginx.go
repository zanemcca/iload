package main

var NGINX_PID int = -1

func reload() {
	//TODO Build the nginx.conf file
	if NGINX_PID > 0 {
		//TODO Compare the old and new nginx.conf files
		//TODO If they are different send the re-conf signal to nginx
		log("Reloading the load balancer")
	} else {
		NGINX_PID = 10
		//TODO Start the load balancer
		log("Starting the load balancer")
	}
}
