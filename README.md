# gdock 

gdock is a gRPC wrapped service which serves functionalities  of [go-dockerclient](github.com/fsouza/go-dockerclient). 
gdock does not include all functionalities of the given library, it means that any function which might be required 
could be added. 

### How to run 

```bash 
$ docker build -t gdock .
$ docker run -d --net=host -v /var/run/docker.sock:/var/run/docker.sock gdock 
```
Docker socket of the host should be mounted in order to use this gRPC service, since the service is about managing docker containers

### Example Client Calls 

There are few examples under [./grpc/client/main.go](./grpc/client/main.go).

