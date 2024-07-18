package query

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bit-bom/bitbom/pkg"
	"github.com/spf13/cobra"
)

type options struct {
	outputdir string
}

func (o *options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&o.outputdir,
		"output-dir",
		"",
		"specify dir to write the output to",
	)
}

func (o *options) Run(_ *cobra.Command, args []string) error {
	script := strings.Join(args, " ")
	// Get the storage instance (assuming a function GetStorageInstance exists)
	storage := pkg.GetStorageInstance("localhost:6379")

	execute, err := pkg.ParseAndExecute(script, storage)
	if err != nil {
		return err
	}
	// Print dependencies
	for _, depID := range execute.ToArray() {
		node, err := storage.GetNode(depID)
		if err != nil {
			fmt.Println("Failed to get name for ID", depID, ":", err)
			continue
		}
		fmt.Println(node.Type, node.Name)

		if o.outputdir != "" {
			data, err := json.MarshalIndent(node.Metadata, "", "	")
			if err != nil {
				return err
			}
			if _, err := os.Stat(o.outputdir); err != nil {
				return err
			}

			filePath := filepath.Join(o.outputdir, strconv.Itoa(int(node.Id))+".json")
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = file.Write(data)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func New() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:               "query [script]",
		Short:             "Query dependencies and dependents of a project",
		RunE:              o.Run,
		DisableAutoGenTag: true,
	}
	o.AddFlags(cmd)

	return cmd
}
