package cli

import (
	"stripe-invoice-go/internal/service"

	"github.com/spf13/cobra"
)

func NewRootCmd(store service.Store, archiver service.Archiver) *cobra.Command {
	root := &cobra.Command{Use: "stripeinvoice"}
	root.AddCommand(NewImportCmd(store, archiver))
	return root
}
