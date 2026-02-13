//go:build !webui

package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewServeCmd creates a stub serve command when built without the webui tag.
func NewServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard server",
		Long:  "Start the web dashboard server (requires build tag: webui)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("the dashboard is not available in this build.\nRebuild with: go build -tags webui ./cmd/wave")
		},
	}
}
