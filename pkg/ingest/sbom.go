package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/bit-bom/minefield/pkg"
)

// IngestSBOM ingests a SBOM file or directory into the storage backend.
func SBOM(sbomPath string, storage pkg.Storage) error {
	info, err := os.Stat(sbomPath)
	if err != nil {
		return fmt.Errorf("error accessing path %s: %w", sbomPath, err)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(sbomPath)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", sbomPath, err)
		}
		for _, entry := range entries {
			entryPath := filepath.Join(sbomPath, entry.Name())
			if err := SBOM(entryPath, storage); err != nil {
				return fmt.Errorf("failed to ingest SBOM from path %s: %w", entryPath, err)
			}
		}
	} else {
		return processSBOMFile(sbomPath, storage)
	}

	return nil
}

// processSBOMFile processes a SBOM file and adds it to the storage backend.
func processSBOMFile(filePath string, storage pkg.Storage) error {
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}

	_, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	bom := new(cdx.BOM)
	decoder := cdx.NewBOMDecoder(file, cdx.BOMFileFormatJSON)
	if err = decoder.Decode(bom); err != nil {
		return fmt.Errorf("failed to decode BOM: %w", err)
	}

	mainBomNode := bom.Metadata.Component

	mainPurl := mainBomNode.PackageURL

	if mainPurl == "" {
		mainPurl = fmt.Sprintf("pkg:generic/%s@%s", mainBomNode.Name, mainBomNode.Version)
	}

	mainGraphNode, err := pkg.AddNode(storage, string(mainBomNode.Type), bom, mainPurl)

	if err != nil {
		return fmt.Errorf("failed to parse SBOM file %s: %w", filePath, err)
	}

	for _, node := range *bom.Components {

		directDep := false
		if node.Properties != nil {
			for _, property := range *node.Properties {
				if strings.Contains(property.Name, "indirect") && property.Value == "false" {
					directDep = true
				}
			}
		}

		if !directDep {
			continue
		}
		purl := node.PackageURL

		if purl == "" {
			purl = fmt.Sprintf("pkg:generic/%s@%s", node.Name, node.Version)
		}

		graphNode, err := pkg.AddNode(storage, string(node.Type), any(node), purl)
		if err != nil {
			if errors.Is(err, pkg.ErrNodeAlreadyExists) {
				// TODO: Add a logger
				fmt.Println("Skipping...")
				continue
			}
			return fmt.Errorf("failed to add node: %w", err)
		}

		if err := mainGraphNode.SetDependency(storage, graphNode); err != nil {
			return fmt.Errorf("failed to add dependencies: %w", err)
		}

	}

	return nil
}
