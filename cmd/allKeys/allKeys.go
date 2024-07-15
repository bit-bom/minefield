package allKeys

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/bit-bom/bitbom/pkg"
)

type options struct{}

func (o *options) AddFlags(_ *cobra.Command) {}

func (o *options) Run(_ *cobra.Command, _ []string) error {
	storage := pkg.GetStorageInstance("localhost:6379")

	keys, err := storage.GetAllKeys()
	if err != nil {
		return fmt.Errorf("failed to query keys: %w", err)
	}

	// Print dependencies
	for _, key := range keys {
		name, err := storage.IDToName(key)
		if err != nil {
			fmt.Println("Failed to get name for ID:", err)
			continue
		}
		fmt.Println(name)
	}

	return nil
}

func New() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:               "keys",
		Short:             "returns all keys",
		RunE:              o.Run,
		DisableAutoGenTag: true,
	}
	o.AddFlags(cmd)

	return cmd
}