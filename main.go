package main

import "github.com/tutumcloud/go-tutum/tutum"
import _ "crypto/sha512"
import "fmt"
import "strings"
import "encoding/json"

//import "reflect"

func log(obj interface{}) {
	//fmt.Printf("%+v\n", obj)
	str, _ := json.MarshalIndent(obj,"", "  ")
	fmt.Printf(string(str))
	fmt.Println()
}

func start() {
	nginxReload()
}

func reload(e tutum.Event) {

	uri := strings.Split(e.Resource_uri, "/")

	ln := len(uri)

	if ln > 2 && uri[len(uri) -3] == "container" {
		container, err := tutum.GetContainer(uri[len(uri) -2])

		if err != nil {
			log(err)
		}

		log(container)

		//TODO Read the new or terminated container and get their IP address
		nginxReload()
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
			reload(event)
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
	start()
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
