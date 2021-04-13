package client

import (
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"neo/pkg/filestream"

	"google.golang.org/grpc"

	neopb "neo/lib/genproto/neo"
)

func New(cc grpc.ClientConnInterface, id string) *Client {
	return &Client{
		c:  neopb.NewExploitManagerClient(cc),
		ID: id,
	}
}

type Client struct {
	c      neopb.ExploitManagerClient
	ID     string
	Weight int
}

func (nc *Client) Ping(ctx context.Context, t neopb.PingRequest_PingType) (*neopb.ServerState, error) {
	req := &neopb.PingRequest{ClientId: nc.ID, Type: t}
	if t == neopb.PingRequest_HEARTBEAT {
		req.Type = neopb.PingRequest_HEARTBEAT
		req.Weight = int32(nc.Weight)
	}
	resp, err := nc.c.Ping(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("making ping request: %w", err)
	}
	return resp.GetState(), nil
}

func (nc *Client) ExploitConfig(ctx context.Context, id string) (*neopb.ExploitConfiguration, error) {
	req := &neopb.ExploitRequest{
		ExploitId: id,
	}
	resp, err := nc.c.Exploit(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("making exploit request: %w", err)
	}
	return resp.GetConfig(), nil
}

func (nc *Client) UpdateExploit(ctx context.Context, req *neopb.UpdateExploitRequest) error {
	if _, err := nc.c.UpdateExploit(ctx, req); err != nil {
		return fmt.Errorf("aking update exploit request: %w", err)
	}
	return nil
}

func (nc *Client) DownloadFile(ctx context.Context, info *neopb.FileInfo, out io.Writer) error {
	resp, err := nc.c.DownloadFile(ctx, info)
	if err != nil {
		return fmt.Errorf("making download file request: %w", err)
	}
	if err := filestream.Save(resp, out); err != nil {
		return fmt.Errorf("saving downloaded file: %w", err)
	}
	return resp.CloseSend()
}

func (nc *Client) UploadFile(ctx context.Context, r io.Reader) (*neopb.FileInfo, error) {
	client, err := nc.c.UploadFile(ctx)
	if err != nil {
		return nil, fmt.Errorf("making upload file request: %w", err)
	}
	if err := filestream.Load(r, client); err != nil {
		return nil, fmt.Errorf("loading filestream: %w", err)
	}
	return client.CloseAndRecv()
}

func (nc *Client) BroadcastCommand(ctx context.Context, command string) error {
	req := &neopb.Command{Command: command}
	if _, err := nc.c.BroadcastCommand(ctx, req); err != nil {
		return fmt.Errorf("making broadcast command request: %w", err)
	}
	return nil
}

func (nc *Client) ListenBroadcasts(ctx context.Context) (chan<- *neopb.Command, error) {
	req := &neopb.Empty{}
	stream, err := nc.c.BroadcastRequests(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating broadcast requests stream: %w", err)
	}

	results := make(chan *neopb.Command)
	go func() {
		commands := make(chan *neopb.Command)
		go func() {
			for {
				cmd, err := stream.Recv()
				if err != nil {
					logrus.Errorf("Error reading from broadcasts channel: %v", err)
					close(commands)
					return
				}
				commands <- cmd
			}
		}()

		for {
			select {
			case cmd := <-commands:
				logrus.Infof("Received a new command from broadcast: %v", cmd)
				results <- cmd
			case <-ctx.Done():
				logrus.Infof("Shutting down broadcast listener")
				close(results)
				return
			}
		}
	}()

	return results, nil
}
