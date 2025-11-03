package sync

import "context"

type Client interface {
	Push(ctx context.Context) error
	Pull(ctx context.Context) error
}

type client struct{}

func New() Client { return &client{} }

func (c *client) Push(ctx context.Context) error { return nil }
func (c *client) Pull(ctx context.Context) error { return nil }
