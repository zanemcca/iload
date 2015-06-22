package main

import "io/ioutil"
import "os/exec"
import "os"
import "strconv"
import "strings"
import "bytes"
import "net"

var upstream string

func getNginxPID() int {
	dat, err := ioutil.ReadFile("/run/nginx.pid")
	if err != nil {
		return -1
	} else {
		str := strings.TrimSpace(string(dat))
		pid, err := strconv.Atoi(str)
		if err != nil {
			log("Error: Could not convert " + str + " to an int")
			return -1
		}
		return pid
	}
}

var backup []byte

func setUpstream(s string) bool {

	const file = "/etc/nginx/servers.conf"

	// If the file exists read a backup then delete the file
	if _, err := os.Stat(file); !os.IsNotExist(err) {

		backup, err := ioutil.ReadFile(file)
		_ = backup

		if err != nil {
			log("Warning: " + file + " could not be read")
			log(err)
		}
		err = os.Remove(file)
		if err != nil {
			log("Error: Could not remove the old file")
			log(err)
			return false
		}
	}

	// Write the new contents to the file
	err := ioutil.WriteFile(file, []byte(s), 0644)
	if err != nil {
		log("Error: Failed to write servers.conf")
		log(err)
		// Since we failed try to restore servers.conf
		err = ioutil.WriteFile(file, backup, 0644)
		if err != nil {
			log("Error: Failed to restore servers.conf")
			log(err)
		}
		return false
	} else {
		upstream = s
		log("Success")
		return true
	}
}

func buildUpstream() string {

	// Grab the hosts 
	hosts, err := net.LookupHost("web")
	if err != nil {
		log(err)
		return upstream
	}

	// See if custom backend ports are being used
	portStr := os.Getenv("BACKEND_PORTS")
	var ports []string

	if len(portStr) > 0 {
		ports = strings.Split(portStr, ",")
	} else {
		ports = append(ports, "80")
	}

	var newUpstream = "upstream servers {"

	// See if a custom load balancing algorithm was asked for
	alg := os.Getenv("BALANCE")
	if newVector("least_conn","ip_hash").contains(alg) {
	  newUpstream += "\n\t" + alg + ";"
	}

	// Add all of the hosts found on all of the ports given
	for _, host := range hosts {
		for _, port := range ports {
			newUpstream += "\n\tserver " + string(host) + ":" + strings.TrimSpace(port) + ";"
		}
	}
	newUpstream += "\n}"

	return newUpstream
}

func nginxReload() {

	newUpstream := buildUpstream()
	log(newUpstream)

	if getNginxPID() > 0 {
		if newUpstream != upstream {
			if setUpstream(newUpstream) {
				log("Reloading the load balancer")
				reload := exec.Command("nginx", "-s", "reload")
				var out bytes.Buffer
				var stderr bytes.Buffer
				reload.Stdout = &out
				reload.Stderr = &stderr

				err := reload.Run()
				if err != nil {
					log("Error: Nginx has failed to reload!")
					log(err)
					log(stderr.String())
				}
				//log(out.String())
			}
		} else {
			log("No need to relod because upstream is identical")
		}
	} else {
		if setUpstream(newUpstream) {
			start := exec.Command("nginx")
			var out bytes.Buffer
			var stderr bytes.Buffer
			start.Stdout = &out
			start.Stderr = &stderr
			log("Starting the load balancer")
			err := start.Start()
			if err != nil {
				log("Error: Nginx has failed to start!")
				log(err)
				log(stderr.String())
			}
			//log(out.String())
		}
	}
	//TODO Add a listener on the output
	//TODO Add a wait task on nginx start command so that we know when it exists
}
