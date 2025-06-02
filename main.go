package main

import (
	"context"
	"resource/cmd"
)

func main() {
	ctx := context.Background()
	cmd.Execute(ctx)
}
