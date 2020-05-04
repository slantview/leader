package election

// Config is our configuration data sent to initialize the backend.
type Config struct {
	NodeID      string
	NodePort    int
	ServiceHost string
	ServicePort int
}
