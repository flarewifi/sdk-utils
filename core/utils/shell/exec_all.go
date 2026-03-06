package shell

import (
	"context"
	"time"
)

func ExecAll(commands []string) error {
	for _, c := range commands {
		err := Exec(c, nil)
		if err != nil {
			return err
		}
		time.Sleep(50 * time.Millisecond) // add delay between commands to prevent potential issues
	}
	return nil
}

// ExecAllWithContext executes multiple commands with context cancellation
func ExecAllWithContext(ctx context.Context, commands []string) error {
	for _, command := range commands {
		if err := ExecWithContext(ctx, command, nil); err != nil {
			return err
		}

		// Check if context was cancelled between commands
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(50 * time.Millisecond) // add delay between commands to prevent potential issues
	}
	return nil
}
