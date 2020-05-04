package election

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
)

// Etcd is an implementation of Election interface for etcd backend.
type Etcd struct {
	config        *Config
	client        *clientv3.Client
	session       *concurrency.Session
	election      *concurrency.Election
	isLeader      bool
	leaderAddress string
	url           string
}

// NewEtcd creates a new Etcd backend.
func NewEtcd(config *Config) *Etcd {
	return &Etcd{
		config:        config,
		isLeader:      false,
		leaderAddress: fmt.Sprintf("http://127.0.0.1:%d", config.NodePort),
		url:           fmt.Sprintf("http://127.0.0.1:%d", config.NodePort),
	}
}

// Init handles initialization and starts the polling.
func (e *Etcd) Init() error {
	// Get a new client to the backend.
	if err := e.getClient(); err != nil {
		return err
	}

	// Connect to the service, register and start updater.
	e.connect()

	// Watch for session end and update the current leader.
	go e.run()

	return nil
}

// IsLeader fufills the Election interface.
func (e *Etcd) IsLeader() bool {
	return e.isLeader
}

// GetLeader fufills the Election interface.
func (e *Etcd) GetLeader() string {
	return e.leaderAddress
}

func (e *Etcd) connect() error {
	// Update the session id for the keepalive.
	if err := e.getSession(); err != nil {
		return err
	}

	// Start our election.
	e.election = concurrency.NewElection(e.session, LeaderNamespace)

	go func() {
		// Attempt to become leader.
		if err := e.election.Campaign(context.Background(), e.url); err != nil {
			fmt.Printf("campaign error: %v\n.", err)
		}
	}()

	// Acquire leadership to determine if we are leader or not.
	if err := e.checkLeader(); err != nil {
		fmt.Printf("error getting leader: %v", err)
	}

	return nil
}

func (e *Etcd) getClient() error {
	var err error
	e.client, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("%s:%d", e.config.ServiceHost, e.config.ServicePort)},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) getSession() error {
	var err error
	e.session, err = concurrency.NewSession(e.client)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) checkLeader() error {
	var err error

	// Get the current leader of the election.
	if e.election == nil {
		fmt.Printf("election not ready.")
		return nil
	}

	leader, err := e.election.Leader(context.Background())
	if err != nil {
		if err == concurrency.ErrElectionNoLeader {
			fmt.Printf("No Leader.")
			return nil
		}
		return err
	}

	// If the leader is the same as our url, then we are leader.
	fmt.Printf("Found leader: %s.\n", leader.Kvs[0].Value)
	e.leaderAddress = string(leader.Kvs[0].Value)
	if e.leaderAddress == e.url {
		e.isLeader = true
	}

	return nil
}

func (e *Etcd) run() {
	updateChan := e.election.Observe(context.Background())
	for {
		select {
		case elect := <-updateChan:
			e.leaderAddress = string(elect.Kvs[0].Value)
			if e.leaderAddress == e.url {
				e.isLeader = true
			}
			fmt.Printf("Updated Leader: %s.\n", e.leaderAddress)
		}
	}
}
