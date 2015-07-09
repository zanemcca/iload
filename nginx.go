package main

import "io/ioutil"
import "io"
import "bufio"
import "os/exec"
import "os"
import "strconv"
import "syscall"
import "strings"
import "bytes"
import "fmt"
import "text/template"
import "reflect"

//import "net"

/*
type Conf struct {
	server   string
	proxy    string
	sslProxy string
}
*/
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
			shutdown(2)
			return -1
		}
		return pid
	}
}

/*
 * Check if nginx is running already
 */
func isNginxRunning() bool {
	pid := getNginxPID()

	if pid > 0 {
		process, err := os.FindProcess(pid)
		if err != nil {
			log("Failed to fine process")
			return false
		} else {
			err := process.Signal(syscall.Signal(0))
			if err != nil {
				fmt.Printf("Signal on pid %d returned: %v\n", pid, err)
				return false
			} else {
				return true
			}
		}
	} else {
		return false
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

func readTemplate() string {
	backup, err := ioutil.ReadFile("nginx/servers.tmpl")
	_ = backup

	if err != nil {
		fmt.Println("Warning: servers.tmpl could not be read")
		panic(err)
	}

	return string(backup)
}

/*
 * Write the configurations to their files
 * then save a local copy in conf
 */
func setConf(c Conf) bool {

	tmpl, err := template.New("servers").Parse(readTemplate())
	if err != nil {
		log(err)
		return false
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, c)

	if err != nil {
		log(err)
		return false
	}

	s:= b.String()
	if safeWrite("nginx/servers.conf",s) {
		log(s)
		conf = c
	}

	return true
}

/*
 * Generate configuration files and start/reload Nginx
 */
func nginxReload(newConf Conf) {

	if isNginxRunning() {
		if ! reflect.DeepEqual(newConf,conf) {
			log(newConf)
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
					shutdown(1)
				}
				log(out.String())
			} else {
				log("Failed to update the configuration")
			}
		} else {
			log("No need to relod because conf is identical")
		}
	} else {
		log(newConf)
		if setConf(newConf) {
			start := exec.Command("nginx")
			stdout, err := start.StdoutPipe()
			if err != nil {
				log(err)
			} else {
				go pipeOutput(stdout)
			}
			stderr, err := start.StderrPipe()
			if err != nil {
				log(err)
			} else {
				go pipeOutput(stderr)
			}

			log("Starting the load balancer")
			err = start.Start()
			if err != nil {
				log("Error: Nginx has failed to start!")
				log(err)
				//log(stderr.String())
				shutdown(1)
			}
			//go stopListener(start)
		} else {
			log("Failed to update the configuration")
		}
	}
}

/*
 * This function will continuously scan and pipe
 * the input to stdout
 */
// Make sure you call this with a go routine
func pipeOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		log("Nginx: " + scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log("Error piping to output!")
		log(err)
	}
}

func shutdown(code int) {
	log("Terminating the instance with code: " + strconv.Itoa(code))
	if isNginxRunning() {
		log("Shutting down the load balancer")
		stop := exec.Command("nginx", "-s", "stop")
		var out bytes.Buffer
		var stderr bytes.Buffer
		stop.Stdout = &out
		stop.Stderr = &stderr

		err := stop.Run()
		if err != nil {
			log("Error: Nginx has failed to stop!")
			log(err)
			log(stderr.String())
		}
		log(out.String())
	}
	os.Exit(code)
}

/*
func stopListener(cmd *exec.Cmd) {
	cmd.Wait()
	log(cmd.Stdout)
	log(cmd.Stderr)
	shutdown(0)
}
*/
