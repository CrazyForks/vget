package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// TakeoutSession manages a Telegram takeout session for bulk downloads
// with lower rate limits. Takeout is Telegram's official data export feature.
type TakeoutSession struct {
	api       *tg.Client
	takeoutID int64
}

// NewTakeoutSession creates a new takeout session manager
func NewTakeoutSession(api *tg.Client) *TakeoutSession {
	return &TakeoutSession{api: api}
}

// Start initiates a takeout session with Telegram
// This enables lower flood wait limits for bulk downloads
func (t *TakeoutSession) Start(ctx context.Context) error {
	req := &tg.AccountInitTakeoutSessionRequest{
		Files:       true,
		FileMaxSize: 4 * 1024 * 1024 * 1024, // 4GB
	}
	req.SetFlags()

	session, err := t.api.AccountInitTakeoutSession(ctx, req)
	if err != nil {
		return fmt.Errorf("init takeout session: %w", err)
	}

	t.takeoutID = session.ID
	return nil
}

// Finish closes the takeout session
func (t *TakeoutSession) Finish(ctx context.Context) error {
	if t.takeoutID == 0 {
		return nil
	}

	req := &tg.AccountFinishTakeoutSessionRequest{Success: true}
	req.SetFlags()

	_, err := t.api.AccountFinishTakeoutSession(ctx, req)
	if err != nil {
		return fmt.Errorf("finish takeout session: %w", err)
	}

	t.takeoutID = 0
	return nil
}

// ID returns the current takeout session ID
func (t *TakeoutSession) ID() int64 {
	return t.takeoutID
}

// Active returns true if a takeout session is active
func (t *TakeoutSession) Active() bool {
	return t.takeoutID != 0
}

// takeoutMiddleware wraps requests with the takeout session ID
type takeoutMiddleware struct {
	id int64
}

// nopDecoder is used to wrap the encoder for InvokeWithTakeoutRequest
type nopDecoder struct {
	bin.Encoder
}

func (n nopDecoder) Decode(_ *bin.Buffer) error {
	return fmt.Errorf("decode not implemented")
}

// Handle implements telegram.Middleware
func (t takeoutMiddleware) Handle(next tg.Invoker) telegram.InvokeFunc {
	return func(ctx context.Context, input bin.Encoder, output bin.Decoder) error {
		return next.Invoke(ctx, &tg.InvokeWithTakeoutRequest{
			TakeoutID: t.id,
			Query:     nopDecoder{input},
		}, output)
	}
}

// Middleware returns a telegram.Middleware that wraps all requests with takeout
func (t *TakeoutSession) Middleware() telegram.Middleware {
	return takeoutMiddleware{id: t.takeoutID}
}
