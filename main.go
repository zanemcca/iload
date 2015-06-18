package main

import "github.com/tutumcloud/go-tutum/tutum"
import _ "crypto/sha512"
import "fmt"

//import "reflect"

func log(obj interface{}) {
	fmt.Printf("%+v\n", obj)
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
	reload();
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
