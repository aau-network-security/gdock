package main

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/mrturkmencom/gdock/config"
	"github.com/mrturkmencom/gdock/docker"
	pb "github.com/mrturkmencom/gdock/docker/proto"
	"google.golang.org/grpc/reflection"
)

func main() {

	if err := config.ValidateConfigPath("/app/config.yml"); err != nil {
		panic(err)
	}
	conf, err := config.NewConfig("/app/config.yml")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Config values endpoint %s, port %s\n", conf.Docker.Domain.Endpoint, conf.Docker.Domain.Port)
	gRPCPort := strconv.FormatUint(uint64(conf.Docker.Domain.Port), 10)

	lis, err := net.Listen("tcp", ":"+gRPCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	dockerService, err := docker.InitDockerService(conf)
	if err != nil {
		return
	}
	opts, err := docker.SecureConn(conf)
	if err != nil {
		log.Fatalf("failed to retrieve secure options %s", err.Error())
	}
	gRPCEndpoint := dockerService.AddAuth(opts...)
	reflection.Register(gRPCEndpoint)
	pb.RegisterDockerServer(gRPCEndpoint, dockerService)

	fmt.Printf("DockerService gRPC server is running at port %s...\n", gRPCPort)
	if err := gRPCEndpoint.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
