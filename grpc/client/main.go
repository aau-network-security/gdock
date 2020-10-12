package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"
	pb "github.com/mrturkmencom/gdock/docker/proto"
	"google.golang.org/grpc"
)

type Creds struct {
	Token    string
	Insecure bool
}

func (c Creds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"token": string(c.Token),
	}, nil
}

func (c Creds) RequireTransportSecurity() bool {
	return !c.Insecure
}

func main() {
	var conn *grpc.ClientConn
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"dockerservice": "test",
	})

	tokenString, err := token.SignedString([]byte("test"))
	if err != nil {
		fmt.Println("Error creating the token")
	}

	authCreds := Creds{Token: tokenString}
	dialOpts := []grpc.DialOption{}
	authCreds.Insecure = true
	dialOpts = append(dialOpts,
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(authCreds))

	conn, err = grpc.Dial(":4444", dialOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	ctx := context.Background()
	cli := pb.NewDockerClient(conn)

	resp, err := cli.Create(ctx, &pb.CreateDockerRequest{
		Image: "guacamole/guacd:1.0.0",
		Labels: map[string]string{
			"hkn": "guacamole_guacd",
		},
		Resource: &pb.CreateDockerRequestResources{
			MemoryMB: 50,
			Cpu:      1.0,
		},
		UseBridge: true,
	})
	if err != nil {
		log.Printf("Error in creating docker container err %v", err)
	}
	fmt.Printf("Create Container Response %s \n", resp.Msg)

	cid := resp.Container.Id
	startResp, err := cli.Start(ctx, &pb.StartDockerRequest{Id: cid})
	if err != nil {
		log.Printf("Error in starting docker container err %v", err)
	}
	fmt.Printf("Start Container Response %s \n", startResp.Msg)

	closeResp, err := cli.Close(ctx, &pb.CloseDockerRequest{Id: cid})
	if err != nil {
		log.Printf("Error in closing docker container err %v", err)
	}

	fmt.Printf("Close Container Response %s \n", closeResp.Msg)
}
