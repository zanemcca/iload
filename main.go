package main

import "github.com/tutumcloud/go-tutum/tutum"
import _ "crypto/sha512"
import "fmt"
import "strings"
import "strconv"
import "os"
import "math/rand"

//import "encoding/json"

func log(obj interface{}) {
	fmt.Printf("%+v\n", obj)
	/*
		str, _ := json.MarshalIndent(obj, "", "  ")
		fmt.Printf(string(str))
		fmt.Println()
	*/
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randName() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type PortMap struct {
	Exposed int
	Local   []int
}

type Service struct {
	Name     string
	Location string
	Hosts    []string
	Auth     bool
}

type Server struct {
	FrontendPort int
	VirtualHost  string
	Services     []Service
}

type Conf struct {
	Servers []Server
}

func findPortMap(maps []PortMap, exposed int) int {
	for i, portMap := range maps {
		if portMap.Exposed == exposed {
			return i
		}
	}
	return -1
}

/*
 * This method will find a service on the server and return a reference to it
 */
func (s *Server) findService(location string) int {
	if len(location) > 0 {
		for i, service := range s.Services {
			if location == service.Location {
				return i
			}
		}
	} else {
		log("Warning: The location given was empty!")
	}
	return -1
}

/*
 * This method will add a service to a server
 */
func (s *Server) addService(service Service) {
	if s.findService(service.Location) >= 0 {
		log("Error: The service already exists on the server. Cannot overwrite it")
		log("	Please use the findService method and modify by reference")
	} else {
		s.Services = append(s.Services, service)
	}
}

/*
 * This method will find any servers on the configuration based on the
 * frontendPort and virtualHost names
 * Both arguments must be passed in for this to work
 */
func (c *Conf) findServer(frontendPort int, virtualHost string) int {
	if frontendPort > 0 && len(virtualHost) > 0 {
		for i, server := range c.Servers {
			if server.FrontendPort == frontendPort && server.VirtualHost == virtualHost {
				return i
			}
		}
	}
	return -1
}

/*
 * This method will add a new server to the conf as long as that server is unique
 */
func (c *Conf) addServer(server Server) {
	if c.findServer(server.FrontendPort, server.VirtualHost) >= 0 {
		log("Error: The server given is not unique and cannot replace the existing one")
		log("	Please use findServer to retrieve a Server reference and modify through thatreference")
	} else {
		c.Servers = append(c.Servers, server)
	}
}

func reload() {

	serviceApi := os.Getenv("TUTUM_SERVICE_API_URI")

	uri := strings.Split(serviceApi, "/")

	ln := len(uri)

	if ln > 2 && uri[len(uri)-3] == "service" {

		service, err := tutum.GetService(uri[len(uri)-2])

		if err != nil {
			log("Error: Failed to retrieve the service")
			log(err)
		}

		containers, err := tutum.ListContainers()

		if err != nil {
			log("Error: Failed to retrieve the list of containers")
			log(err)
		}

		conf := Conf{}

		for _, link := range service.Linked_to_service {
			first := true
			var maps []PortMap
			location := "/"
			vhost := "localhost"
			auth := false

			for _, container := range containers.Objects {
				if container.State == "Running" && container.Service == link.To_service {

					//Some stuff only has to be run once per container of each service
					if first {
						first = false

						//The container returned by getContainers skips the env vars
						// so I am trying to re-retrieve the container individually
						tempContainer, err := tutum.GetContainer(container.Uuid)
						if err != nil {
							log("Error: Failed to get the container: " + container.Uuid)
							log(err)
							os.Exit(5)
						} else {
							container = tempContainer
						}

						//Check if the container requests custom backend ports
						for _, pair := range container.Container_envvars {
							if pair.Key == "PORT_MAP" {
								//Should be of form "exposed:local,exposed:local"
								value := strings.Split(pair.Value, ",")
								for _, portMap := range value {
									//Should be of form "exposed:local"
									ports := strings.Split(portMap, ":")
									exposed, err := strconv.Atoi(ports[0])
									if err != nil {
										log("Error: Port '" + ports[0] + "' is not valid for " + link.Name)
										continue
									}

									local, err := strconv.Atoi(ports[1])
									if err != nil {
										log("Error: Port '" + ports[1] + "' is not valid for " + link.Name)
										continue
									}
									ind := findPortMap(maps, exposed)
									if ind < 0 {
										maps = append(maps, PortMap{Exposed: exposed, Local: []int{local}})
									} else {
										maps[ind].Local = append(maps[ind].Local, local)
									}
								}
							} else if pair.Key == "LOCATION" {
								location = pair.Value
							} else if pair.Key == "VIRTUAL_HOST" {
								vhost = pair.Value
							} else if pair.Key == "HTTP_AUTH" {
								auth = true
								/*
									value := strings.Split(pair.Value, ":")
										if len(value) == 2 {
											auth = true
										} else {
											log("Warning: The http authentication is no good")
											log("	It is expected to be of the form 'username:pasword'")
										}
								*/
							}
						}
					}

					if len(maps) > 0 {
						//If custom backend ports were requested then load them
						for _, mMap := range maps {
							i := conf.findServer(mMap.Exposed, vhost)
							if i < 0 {
								server := Server{FrontendPort: mMap.Exposed, VirtualHost: vhost}
								conf.addServer(server)
								i = conf.findServer(mMap.Exposed, vhost)
							}
							j := conf.Servers[i].findService(location)
							if j < 0 {
								service := Service{Location: location, Auth: auth, Name: link.Name + randName()}
								conf.Servers[i].addService(service)
								j = conf.Servers[i].findService(location)
							}

							for _, port := range mMap.Local {
								address := container.Private_ip + ":" + strconv.Itoa(port)
								conf.Servers[i].Services[j].Hosts = append(conf.Servers[i].Services[j].Hosts, address)
							}
						}
					} else {

						i := conf.findServer(80, vhost)
						if i < 0 {
							server := Server{FrontendPort: 80, VirtualHost: vhost}
							conf.addServer(server)
							i = conf.findServer(80, vhost)
						}

						j := conf.Servers[i].findService(location)
						if j < 0 {
							service := Service{Location: location, Auth: auth, Name: link.Name + randName()}
							conf.Servers[i].addService(service)
							j = conf.Servers[i].findService(location)
						}

						for _, port := range container.Container_ports {
							port_num := strconv.Itoa(port.Inner_port)
							address := container.Private_ip + ":" + port_num
							conf.Servers[i].Services[j].Hosts = append(conf.Servers[i].Services[j].Hosts, address)
						}
					}
				}
			}
		}

		log("Info: Configuration is completed. Reloading Nginx")
		nginxReload(conf)

	} else {
		log("Error: The service URI is not valid")
		log(uri)
		os.Exit(4)
	}

}

func eventHandler(event tutum.Event) {

	log("Event:")
	log(event)

	notState := newVector("In progress",
		"Pending",
		"Terminating",
		"Starting",
		"Scaling",
		"Stopping")

	//types := newVector("container", "service")
	types := newVector("container")

	log("Info: Vectors created!")
	if !notState.contains(event.State) {
		log("Info: Valid State!")
		if types.contains(event.Type) {
			log("Info: Valid Type! Reloading!")
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
			shutdown(-6)
		}
	}
}
