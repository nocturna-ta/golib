package event

import (
	"context"
	"fmt"
)

var (
	commitStrategy = map[string]CommitStrategyFactory{
		"always_commit":     CommitFactory(AlwaysCommitStrategy),
		"commit_on_success": CommitFactory(CommitOnSuccessStrategy),
	}
)

type (
	CommitStrategy        func(ctx context.Context, message ConsumeMessage, handler ConsumerHandler) error
	CommitStrategyFactory func(ctx context.Context, config any) (CommitStrategy, error)
)

// AlwaysCommitStrategy will always commit the message no matter what is the handler result
func AlwaysCommitStrategy(ctx context.Context, message ConsumeMessage, handler ConsumerHandler) error {
	em, err := message.GetEventConsumeMessage(ctx)
	if err != nil {
		return fmt.Errorf("[event/alwaysCommitStrategy] failed to get event message: %w", err)
	}

	// add request context from metadata to existing context
	ctx = requestContextFromMetadata(ctx, em)

	if err := message.Commit(ctx); err != nil {
		return fmt.Errorf("[event/alwaysCommitStrategy] failed to commit: %w", err)
	}

	return handler(ctx, em)
}

// CommitOnSuccessStrategy will only commit the message if handler doesn't return error
func CommitOnSuccessStrategy(ctx context.Context, message ConsumeMessage, handler ConsumerHandler) error {
	em, err := message.GetEventConsumeMessage(ctx)
	if err != nil {
		return fmt.Errorf("[event/commitOnSuccessStrategy] failed to get event message: %w", err)
	}

	// add request context from metadata to existing context
	ctx = requestContextFromMetadata(ctx, em)

	if err := handler(ctx, em); err != nil {
		return fmt.Errorf("[event/commitOnSuccessStrategy] handler failed to process message: %w", err)
	}

	if err := message.Commit(ctx); err != nil {
		return fmt.Errorf("[event/commitOnSuccessStrategy] failed to commit: %w", err)
	}

	return nil
}

func CommitFactory(strategy CommitStrategy) CommitStrategyFactory {
	return func(ctx context.Context, config any) (CommitStrategy, error) {
		return strategy, nil
	}
}
