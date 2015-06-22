package main

import "io/ioutil"
import "os/exec"
import "os"
import "strconv"
import "strings"
import "bytes"

//import "net"

type Conf struct {
	server string
	proxy  string
}

var conf Conf

/*
 * Get the Nginx PID from the /run/nginx.pid file
 */
func getNginxPID() int {
	dat, err := ioutil.ReadFile("/run/nginx.pid")
	if err != nil {
		return -1
	} else {
		str := strings.TrimSpace(string(dat))
		pid, err := strconv.Atoi(str)
		if err != nil {
			log("Error: Could not convert " + str + " to an int")
			os.Exit(2)
			return -1
		}
		return pid
	}
}

/*
 * Safely write the contents to the file
 * A backup is created first and if the write fails then
 * the backup is written back to the file
 */
func safeWrite(filename string, contents string) bool {

	var backup []byte

	// If the file exists read a backup then delete the file
	if _, err := os.Stat(filename); !os.IsNotExist(err) {

		backup, err := ioutil.ReadFile(filename)
		_ = backup

		if err != nil {
			log("Warning: " + filename + " could not be read")
			log(err)
		}
		err = os.Remove(filename)
		if err != nil {
			log("Error: Could not remove the old file")
			log(err)
			return false
		}
	}

	// Write the new contents to the file
	err := ioutil.WriteFile(filename, []byte(contents), 0644)
	if err != nil {
		log("Error: Failed to write servers.conf")
		log(err)
		// Since we failed try to restore servers.conf
		err = ioutil.WriteFile(filename, backup, 0644)
		if err != nil {
			log("Error: Failed to restore servers.conf")
			log(err)
		}
		return false
	} else {
		return true
	}
}

/*
 * Write the configurations to their files
 * then save a local copy in conf
 */
func setConf(c Conf) bool {

	const server = "/etc/nginx/servers.conf"
	const proxy = "/etc/nginx/proxy.conf"

	success := true
	if c.server != conf.server {
		success = safeWrite(server, c.server) && success
	}
	if c.proxy != conf.proxy {
		success = safeWrite(proxy, c.proxy) && success
	}

	if success {
		conf = c
	} else {
	  os.Exit(3)
	}

	return success
}

/*
 * Build the configurations using the service name and
 * corresponding addresses
 */
func buildConf(services []ServiceAddrs) Conf {

	var newConf Conf
	for _, service := range services {
		newConf.proxy += "proxy_pass http://" + service.name + ";"

		var ustream = "upstream " + service.name + " {"

		// See if a custom load balancing algorithm was asked for
		alg := os.Getenv("BALANCE")
		if newVector("least_conn", "ip_hash").contains(alg) {
			ustream += "\n\t" + alg + ";"
		}

		for _, adr := range service.addrs {

			ustream += "\n\tserver " + adr + ";"
		}
		ustream += "\n}"
		newConf.server += ustream + "\n"
	}

	return newConf
}

/*
 * Generate configuration files and start/reload Nginx
 */
func nginxReload(services []ServiceAddrs) {

	newConf := buildConf(services)
	log(newConf)

	if getNginxPID() > 0 {
		if newConf != conf {
			if setConf(newConf) {
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
					os.Exit(1)
				}
				//log(out.String())
			}
		} else {
			log("No need to relod because conf is identical")
		}
	} else {
		if setConf(newConf) {
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
				os.Exit(1)
			}
			go stopListener(start)
		}
	}
}

func stopListener(cmd *exec.Cmd) {
	cmd.Wait()
	log(cmd.Stdout)
	log(cmd.Stderr)
	os.Exit(1)
}
