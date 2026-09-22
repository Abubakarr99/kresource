package main

import (
	"context"
	"github.com/Abubakarr99/kresource/cmd"
)

func main() {
	ctx := context.Background()
	cmd.Execute(ctx)
}
