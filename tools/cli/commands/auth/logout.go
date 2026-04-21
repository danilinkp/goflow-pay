package auth

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func NewLogoutCmd(httpClient *client.Client) *cobra.Command {
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from GoFlow Pay",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := httpClient.Post(context.Background(), "/api/v1/auth/logout", nil, nil); err != nil {
				return fmt.Errorf("logout failed: %w", err)
			}

			if err := credentials.Delete(); err != nil {
				return fmt.Errorf("failed to delete credentials: %w", err)
			}

			fmt.Println("Logged out successfully")
			return nil
		},
	}

	return logoutCmd
}
