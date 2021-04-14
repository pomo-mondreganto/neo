package client

import (
	"context"
	"errors"
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
	if err := resp.CloseSend(); err != nil {
		return fmt.Errorf("closing the stream: %w", err)
	}
	return nil
}

func (nc *Client) UploadFile(ctx context.Context, r io.Reader) (*neopb.FileInfo, error) {
	client, err := nc.c.UploadFile(ctx)
	if err != nil {
		return nil, fmt.Errorf("making upload file request: %w", err)
	}
	if err := filestream.Load(r, client); err != nil {
		return nil, fmt.Errorf("loading filestream: %w", err)
	}
	fileInfo, err := client.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("closing & reading upload response: %w", err)
	}
	return fileInfo, nil
}

func (nc *Client) BroadcastCommand(ctx context.Context, command string) error {
	req := &neopb.Command{Command: command}
	if _, err := nc.c.BroadcastCommand(ctx, req); err != nil {
		return fmt.Errorf("making broadcast command request: %w", err)
	}
	return nil
}

func (nc *Client) SingleRun(ctx context.Context, exploitID string) error {
	req := &neopb.ExploitRequest{ExploitId: exploitID}
	if _, err := nc.c.SingleRun(ctx, req); err != nil {
		return fmt.Errorf("making broadcast command request: %w", err)
	}
	return nil
}

func (nc *Client) ListenBroadcasts(ctx context.Context) (<-chan *neopb.Command, error) {
	stream, err := nc.c.BroadcastRequests(ctx, &neopb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("creating broadcast requests stream: %w", err)
	}

	results := make(chan *neopb.Command)
	go func() {
		defer close(results)
		for {
			cmd, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				logrus.Errorf("Broadcast stream closed")
				return
			}
			if err != nil {
				logrus.Errorf("Error reading from broadcasts stream: %v", err)
				return
			}
			select {
			case results <- cmd:
			case <-ctx.Done():
				logrus.Warningf("Broadcast context cancelled")
				return
			}
		}
	}()

	return results, nil
}

func (nc *Client) ListenSingleRuns(ctx context.Context) (<-chan *neopb.ExploitRequest, error) {
	stream, err := nc.c.SingleRunRequests(ctx, &neopb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("creating single run requests stream: %w", err)
	}

	results := make(chan *neopb.ExploitRequest)
	go func() {
		defer close(results)
		for {
			er, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				logrus.Errorf("Single runs stream closed by server")
				return
			}
			if err != nil {
				logrus.Errorf("Error reading from single runs stream: %v", err)
				return
			}
			select {
			case results <- er:
			case <-ctx.Done():
				logrus.Warningf("Single runs context cancelled")
				return
			}
		}
	}()

	return results, nil
}
