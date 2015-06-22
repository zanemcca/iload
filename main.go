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
					for _, port := range container.Container_ports {
						port_num := strconv.Itoa(port.Inner_port)
						address := container.Private_ip + ":" + port_num
						srv.addrs = append(srv.addrs, address)
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
			log("Error:")
			log(err)
		}
	}
}
