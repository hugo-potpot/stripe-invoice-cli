package cli

import (
	"stripe-invoice-go/internal/service"

	"github.com/spf13/cobra"
)

func NewRootCmd(store service.Store, archiver service.Archiver, mailer service.Mailer) *cobra.Command {
	root := &cobra.Command{Use: "stripeinvoice"}
	root.AddCommand(NewImportCmd(store, archiver))
	root.AddCommand(NewExportCmd(store, archiver))
	root.AddCommand(NewSendCmd(store, archiver, mailer))
	return root
}
