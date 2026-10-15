package companies

import (
	"cli/internal/client"
	"cli/internal/credentials"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type companyItemResponse struct {
	CompanyId uuid.UUID `json:"company_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewListCompaniesCmd(httpClient *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all companies (System Admin only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := credentials.Load()
			if err != nil {
				return err
			}
			if creds.Role != "admin" {
				return fmt.Errorf("only system admin can list all companies")
			}

			var resp struct {
				Items []companyItemResponse `json:"items"`
				Total int                   `json:"total"`
			}

			if err = httpClient.Get(context.Background(), "/api/v2/companies", &resp); err != nil {
				return fmt.Errorf("failed to list companies: %w", err)
			}

			if len(resp.Items) == 0 {
				fmt.Println("No companies found")
				return nil
			}

			fmt.Printf("Total companies: %d\n", resp.Total)
			for _, c := range resp.Items {
				fmt.Printf("ID: %s | Name: %s | Created: %s\n",
					c.CompanyId, c.Name, c.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}

	return cmd
}
