package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	"neo/pkg/server"

	"google.golang.org/grpc"

	neopb "neo/lib/genproto/neo"
)

var (
	configFile = flag.String("config", "config.yaml", "yaml config file to read")
)

func load(path string, cfg *server.Configuration) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return server.ReadConfig(data, cfg)
}

func main() {
	flag.Parse()
	cfg := &server.Configuration{}
	if err := load(*configFile, cfg); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	st, err := server.NewBoltStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to create bolt storage: %v", err)
	}
	srv := server.New(cfg, st)
	lis, err := net.Listen("tcp", ":" + cfg.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	neopb.RegisterExploitManagerServer(s, srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go srv.HeartBeat(ctx)
	go func() {
		<-c
		cancel()
		s.GracefulStop()
	}()
	logrus.Infof("Starting server on port %s", cfg.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
