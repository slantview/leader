# Leader Election

## Requirements

* Implement a leader-follower distributed system.
* As leader, a server should respond with "hello world" when a GET is called on "/hello".
* As follower, a server should respond with the IP/port of the leader when a GET is called on "/hello".
* If the leader goes down, a follower should become the leader.
* Only one server should be leader at a time.

## Additional Features

* Returns a 302 redirect for leader.
* Has Health Check on /_health.
* Has Etcd backend.
* Has Consul backend.

## Building

* Have Go runtime installed.

```
$ make
```

### Additional Make Commands

* See Makefile for additional commands.

## Running

```
# For consul:
$ ./leader -consul -port 8080

# For etcd:
$ ./leader -etcd -port 8080
```
