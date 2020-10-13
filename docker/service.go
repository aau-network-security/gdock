package docker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/mrturkmencom/gdock/config"
	pb "github.com/mrturkmencom/gdock/docker/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type dockerservice struct {
	auth   Authenticator
	config *config.Config
}

func InitDockerService(conf *config.Config) (*dockerservice, error) {
	ds := dockerservice{
		auth:   NewAuthenticator(conf.Docker.AUTH.SignKey, conf.Docker.AUTH.AuthKey),
		config: conf,
	}
	return &ds, nil
}

func (d *dockerservice) Create(ctx context.Context, req *pb.CreateDockerRequest) (*pb.CreateDockerResponse, error) {
	// initialize docker
	rs := &Resources{
		MemoryMB: uint(req.Resource.MemoryMB),
		CPU:      float64(req.Resource.Cpu),
	}
	containerConfig := ContainerConfig{
		Image:        req.Image,
		EnvVars:      req.EnvVars,
		PortBindings: req.PortBindings,
		Labels:       req.Labels,
		Mounts:       req.Mounts,
		Resources:    rs,
		Cmd:          req.Cmd,
		DNS:          req.Dns,
		UsedPorts:    req.UsedPorts,
		UseBridge:    req.UseBridge,
	}
	c := NewContainer(containerConfig)
	if err := c.Create(ctx); err != nil {
		return &pb.CreateDockerResponse{}, err
	}
	return &pb.CreateDockerResponse{Msg: fmt.Sprintf("Container created with id %s", c.ID()), Container: &pb.CreateDockerResponse_Container{
		Id:    c.ID(),
		State: int32(c.Info().State),
		Image: c.Info().Image,
		Type:  c.Info().Type,
	}}, nil
}
func (d *dockerservice) Start(ctx context.Context, req *pb.StartDockerRequest) (*pb.StartDockerResponse, error) {
	// not sure whether will work or not
	c := &container{
		id: req.Id,
	}
	if err := c.Start(ctx); err != nil {
		return &pb.StartDockerResponse{}, err
	}
	return &pb.StartDockerResponse{Msg: fmt.Sprintf("Container is started with id %s", c.id)}, nil
}
func (d *dockerservice) Suspend(ctx context.Context, req *pb.SuspendDockerRequest) (*pb.SuspendDockerResponse, error) {
	c := &container{
		id: req.Id,
	}
	if err := c.Suspend(); err != nil {
		return &pb.SuspendDockerResponse{}, err
	}
	return &pb.SuspendDockerResponse{Msg: fmt.Sprintf("Container is suspended with id %s", c.id)}, nil
}

func (d *dockerservice) Run(ctx context.Context, req *pb.RunDockerRequest) (*pb.RunDockerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Run not implemented")
}
func (d *dockerservice) Stop(ctx context.Context, req *pb.StopDockerRequest) (*pb.StopDockerResponse, error) {
	c := &container{
		id: req.Id,
	}
	if err := c.Stop(); err != nil {
		return &pb.StopDockerResponse{}, err
	}
	return &pb.StopDockerResponse{Msg: fmt.Sprintf("Container is suspended with id %s", c.id)}, nil
}
func (d *dockerservice) Close(ctx context.Context, req *pb.CloseDockerRequest) (*pb.CloseDockerResponse, error) {
	c := &container{
		id: req.Id,
	}
	if err := c.Close(); err != nil {
		return &pb.CloseDockerResponse{}, err
	}
	return &pb.CloseDockerResponse{Msg: fmt.Sprintf("Container is suspended with id %s", c.id)}, nil
}
func (d *dockerservice) Info(ctx context.Context, req *pb.InfoDockerRequest) (*pb.InfoDockerResponse, error) {
	c := &container{
		id: req.Id,
	}
	return &pb.InfoDockerResponse{Container: &pb.InfoDockerResponse_Container{
		Id:    c.Info().Id,
		State: int32(c.Info().State),
		Image: c.Info().Image,
		Type:  c.Info().Type,
	}}, nil
}

func GetCreds(conf config.Config) (credentials.TransportCredentials, error) {
	log.Printf("Preparing credentials for RPC")

	certificate, err := tls.LoadX509KeyPair(conf.Docker.TLS.CertFile, conf.Docker.TLS.CertKey)
	if err != nil {
		return nil, fmt.Errorf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authorityS
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(conf.Docker.TLS.CAFile)
	if err != nil {
		return nil, fmt.Errorf("could not read ca certificate: %s", err)
	}
	// CA file for let's encrypt is located under domain conf as `chain.pem`
	// pass chain.pem location
	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	// Create the TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})
	return creds, nil
}

// SecureConn enables communication over secure channel
func SecureConn(conf *config.Config) ([]grpc.ServerOption, error) {
	if conf.Docker.TLS.Enabled {
		log.Info().Msgf("Conf cert-file: %s, cert-key: %s ca: %s", conf.Docker.TLS.CertFile, conf.Docker.TLS.CertKey, conf.Docker.TLS.CAFile)
		creds, err := GetCreds(*conf)

		if err != nil {
			return []grpc.ServerOption{}, errors.New("Error on retrieving certificates: " + err.Error())
		}
		log.Printf("Server is running in secure mode !")
		return []grpc.ServerOption{grpc.Creds(creds)}, nil
	}
	return []grpc.ServerOption{}, nil
}

func InitServer(conf *config.Config) (*dockerservice, error) {

	gRPCServer := &dockerservice{
		auth:   NewAuthenticator(conf.Docker.AUTH.SignKey, conf.Docker.AUTH.AuthKey),
		config: conf,
	}
	return gRPCServer, nil
}

// AddAuth adds authentication to gRPC server
func (d *dockerservice) AddAuth(opts ...grpc.ServerOption) *grpc.Server {
	streamInterceptor := func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := d.auth.AuthenticateContext(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}

	unaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := d.auth.AuthenticateContext(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}

	opts = append([]grpc.ServerOption{
		grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(unaryInterceptor),
	}, opts...)
	return grpc.NewServer(opts...)

}
