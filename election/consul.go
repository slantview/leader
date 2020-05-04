package election

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

// Consul is an implementation of Election for Consul backend.
type Consul struct {
	config        *Config
	client        *api.Client
	isLeader      bool
	leaderAddress string
	sessionID     string
	doneChan      chan struct{}
}

// NewConsul creates a new Consul backend.
func NewConsul(config *Config) *Consul {
	return &Consul{
		config:        config,
		isLeader:      false,
		leaderAddress: fmt.Sprintf("http://127.0.0.1:%d", config.NodePort),
	}
}

// Init handles initialization and starts the polling.
func (c *Consul) Init() error {
	// Get a new client to the backend.
	if err := c.getClient(); err != nil {
		return err
	}

	// Connect to the service, register and start updater.
	if err := c.connect(); err != nil {
		return err
	}

	// Watch the session and periodically update to keep session active.
	go c.updateSession()

	// Watch for session end and update the current leader.
	go c.run()

	return nil
}

// IsLeader fufills the Election interface.
func (c *Consul) IsLeader() bool {
	return c.isLeader
}

// GetLeader fufills the Election interface.
func (c *Consul) GetLeader() string {
	return c.leaderAddress
}

func (c *Consul) updateSession() {
	c.doneChan = make(chan struct{})
	c.client.Session().RenewPeriodic(
		"10s",
		c.sessionID,
		nil,
		c.doneChan,
	)
}

func (c *Consul) getClient() error {
	var err error

	c.client, err = api.NewClient(&api.Config{
		Address: fmt.Sprintf("%s:%d", c.config.ServiceHost, c.config.ServicePort),
		Scheme:  "http",
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Consul) registerService() error {
	err := c.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		Address: fmt.Sprintf("http://127.0.0.1:%d", c.config.NodePort),
		ID:      c.config.NodeID,
		Name:    "election",
		Tags:    []string{"election"},
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://127.0.0.1:%d/_health", c.config.NodePort),
			Interval: "10s",
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *Consul) getSession() error {
	var err error
	c.sessionID, _, err = c.client.Session().Create(&api.SessionEntry{
		Name:     LeaderNamespace[1:],
		Behavior: "delete",
		TTL:      "10s",
	}, nil)

	if err != nil {
		return err
	}
	return nil
}

func (c *Consul) checkLeader() error {
	var err error
	c.isLeader, _, err = c.client.KV().Acquire(&api.KVPair{
		Key:     LeaderNamespace[1:],
		Value:   []byte(fmt.Sprintf("http://127.0.0.1:%d", c.config.NodePort)),
		Session: c.sessionID,
	}, nil)

	if err != nil {
		return err
	}

	return nil
}

func (c *Consul) getLeader() error {
	pair, _, err := c.client.KV().Get(LeaderNamespace[1:], nil)
	if err != nil {
		return err
	}

	// If we don't get a pair back, we didn't find a leader.
	if pair != nil {
		fmt.Printf("Found leader at %s.\n", pair.Value)
		c.leaderAddress = string(pair.Value)
		return nil
	}

	// If there isn't a leader anymore, reconnect new session.
	c.connect()

	return nil
}

func (c *Consul) run() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-c.doneChan:
			fmt.Printf("Received stop on done channel.")
			// Once we are done we need to reconnect.
			close(c.doneChan)
			// Reconnect by re-establishing a service.
			c.connect()
			return
		case <-ticker.C:
			if err := c.getLeader(); err != nil {
				fmt.Printf("error: %v\n", err)
			}
		}
	}
}

func (c *Consul) connect() error {
	// Register the Consul service.
	if err := c.registerService(); err != nil {
		return err
	}

	// Update the session id for the keepalive.
	if err := c.getSession(); err != nil {
		return err
	}

	// Acquire leadership to determine if we are leader or not.
	if err := c.checkLeader(); err != nil {
		fmt.Printf("error getting leader: %v", err)
	}

	// Watch the session and periodically update to keep session active.
	go c.updateSession()

	return nil
}
