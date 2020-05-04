package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/slantview/leader/election"
)

var (
	listenPort = flag.Int("port", 8080, "Port to listen on.")
	useConsul  = flag.Bool("consul", false, "Use Consul for backend.")
	consulURL  = flag.String("consul-url", "localhost:8500", "Consul host:port")
	useEtcd    = flag.Bool("etcd", false, "Use Etcd for backend.")
	etcdURL    = flag.String("etcd-url", "localhost:2379", "Etcd host:port")
	version    = "0.1.0"
	hash       = "deadbeef"
)

func init() {
	flag.Parse()
	switch os.Getenv("LEADER_ELECTION_BACKEND") {
	case "etcd":
		*useEtcd = true
		if os.Getenv("LEADER_ELECTION_URL") != "" {
			*etcdURL = os.Getenv("LEADER_ELECTION_URL")
		}
	case "consul":
		*useConsul = true
		if os.Getenv("LEADER_ELECTION_URL") != "" {
			*consulURL = os.Getenv("LEADER_ELECTION_URL")
		}
	}
}

func main() {
	nodeID := uuid.New()
	fmt.Printf("%s v%s-%s starting on port %d.\n", nodeID, version, hash, *listenPort)

	// If neither are set, exit non-zero with an error.
	if !*useConsul && !*useEtcd {
		panic("Must select a backend for leader election.")
	}

	// Setup our backend interface. We will check the environment variables to
	// determine which backend we will use.
	var backend election.Election

	// If we are using Consul for our backend, setup the election backend.
	if *useConsul {
		fmt.Printf("Using Consul on %s.\n", *consulURL)
		np := strings.Split(*consulURL, ":")
		port, err := strconv.Atoi(np[1])
		if err != nil {
			panic("Error in Consul port: " + err.Error())
		}
		backend = election.NewConsul(&election.Config{
			NodeID:      nodeID.String(),
			NodePort:    *listenPort,
			ServiceHost: np[0],
			ServicePort: port,
		})
	}

	// If we are using Etcd for our backend, setup the election backend.
	if *useEtcd {
		fmt.Printf("Using Etcd on %s.\n", *etcdURL)
		np := strings.Split(*etcdURL, ":")
		port, err := strconv.Atoi(np[1])
		if err != nil {
			panic("Error in Etcd port: " + err.Error())
		}
		backend = election.NewEtcd(&election.Config{
			NodeID:      nodeID.String(),
			NodePort:    *listenPort,
			ServiceHost: np[0],
			ServicePort: port,
		})
	}

	// If we can't init our backend due to connection issue or other, we need to
	// bail now. In a container environment, the container should be restarted
	// and this will be eventually consistent.
	if err := backend.Init(); err != nil {
		panic(err)
	}

	// Create a new Server and use our backend Election interface.
	s := NewServer(backend)

	// Create new echo instance to handle our HTTP service. We use echo instead
	// of the default http.ServeMux because it is faster and has much lower
	// memory allocations. See https://github.com/labstack/echo for more info.
	e := echo.New()

	// Register our routes. Root route handles the problem. Our health check
	// route is for Consul to do keepalives on the service to timeout for our
	// leader election.
	e.GET("/hello", s.Hello)
	e.GET("/_health", s.HealthCheck)

	// Start the HTTP interface. This is a blocking action and when we return
	// from here it will always be a runtime error. Panic so we return a
	// non-zero error.
	if err := e.Start(fmt.Sprintf(":%d", *listenPort)); err != nil {
		panic(err)
	}
}
