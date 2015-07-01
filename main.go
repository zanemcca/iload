package main

import "github.com/tutumcloud/go-tutum/tutum"
import _ "crypto/sha512"
import "fmt"
import "strings"

//import "encoding/json"
import "strconv"
import "os"

//import "reflect"

func log(obj interface{}) {
	fmt.Printf("%+v\n", obj)
	/*
		str, _ := json.MarshalIndent(obj, "", "  ")
		fmt.Printf(string(str))
		fmt.Println()
	*/
}

type ServiceAddrs struct {
	name  string   `json:"name"`
	addrs []string `json:"addrs"`
}

func reload() {

	uri := strings.Split(os.Getenv("TUTUM_SERVICE_API_URI"), "/")

	ln := len(uri)

	if ln > 2 && uri[len(uri)-3] == "service" {

		service, err := tutum.GetService(uri[len(uri)-2])

		if err != nil {
			log(err)
		}

		containers, err := tutum.ListContainers()

		if err != nil {
			log(err)
		}

		var addrs []ServiceAddrs

		for _, link := range service.Linked_to_service {
			var srv ServiceAddrs
			srv.name = link.Name
			for _, container := range containers.Objects {
				if container.State == "Running" && container.Service == link.To_service {
					//The container returned by getContainers skips the env vars
					// so I am trying to re-retrieve the container individually
					tempContainer, err := tutum.GetContainer(container.Uuid)
					if err != nil {
						log(err)
						os.Exit(5)
					} else {
						container = tempContainer
					}
					//Check if the container requests custom backend ports
					var ports []string
					for _, pair := range container.Container_envvars {
						if pair.Key == "BACKEND_PORT" || pair.Key == "BACKEND_PORTS" {
							ports = strings.Split(pair.Value, ",")
							break
						}
					}
					if ports != nil {
						//If custom backend ports were requested then load them
						for _, port := range ports {
							_, err := strconv.Atoi(port)
							if err != nil {
								log("Error: Port '" + port + "' is not valid for " + srv.name)
							} else {
								address := container.Private_ip + ":" + port
								srv.addrs = append(srv.addrs, address)
							}
						}
					} else {
						for _, port := range container.Container_ports {
							port_num := strconv.Itoa(port.Inner_port)
							address := container.Private_ip + ":" + port_num
							srv.addrs = append(srv.addrs, address)
						}
					}
				}
			}
			addrs = append(addrs, srv)
		}

		log(addrs)
		nginxReload(addrs)

	} else {
		log("Error: The service URI is not valid")
		log(uri)
		os.Exit(4)
	}

}

func eventHandler(event tutum.Event) {

	notState := newVector("In progress",
		"Pending",
		"Terminating",
		"Starting",
		"Scaling",
		"Stopping")

	//types := newVector("container", "service")
	types := newVector("container")

	if !notState.contains(event.State) {
		if types.contains(event.Type) {
			reload()
		}
	}
}

/* Main function

*/
func main() {
	tutum.User = "zanemcca"
	tutum.ApiKey = "37d751a00f6755575819f12ffc665b3c5aa6d9db"

	c := make(chan tutum.Event)
	e := make(chan error)

	// Launch the load balancer
	reload()
	log("Starting the tutum event handler")

	go tutum.TutumEvents(c, e)

	for {
		select {
		case event := <-c:
			eventHandler(event)
		case err := <-e:
			if err != nil {
				log("Error:")
				log(err)
			}
		}
	}
}
