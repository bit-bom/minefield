package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bit-bom/minefield/pkg"
	"github.com/protobom/protobom/pkg/reader"
	"github.com/protobom/protobom/pkg/sbom"
)

// SBOM ingests a SBOM file or directory into the storage backend.
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
	sbomReader := reader.New()

	document, err := sbomReader.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse SBOM file %s: %w", filePath, err)
	}
	nameToNodeID := map[string]uint32{}

	for _, node := range document.GetNodeList().GetNodes() {
		purl := string(node.Purl())
		if purl == "" {
			purl = fmt.Sprintf("pkg:generic/%s@%s", node.Name, node.Version)
		}

		graphNode, err := pkg.AddNode(storage, node.Type.String(), any(node), purl)
		if err != nil {
			if errors.Is(err, pkg.ErrNodeAlreadyExists) {
				// TODO: Add a logger
				fmt.Println("Skipping...")
				continue
			}
			return fmt.Errorf("failed to add node: %w", err)
		}
		nameToNodeID[purl] = graphNode.ID
	}

	err = addDependency(document, storage, nameToNodeID)
	if err != nil {
		return fmt.Errorf("failed to add dependencies: %w", err)
	}

	return nil
}

// addDependency iterates over all the edges protobom sbom document and creates a dependency edge between each node in an edge
func addDependency(document *sbom.Document, storage pkg.Storage, nameToNodeID map[string]uint32) error {
	for _, edge := range document.GetNodeList().GetEdges() {
		fromProtoNode := document.GetNodeList().GetNodeByID(edge.From)
		fromPurl := string(fromProtoNode.Purl())
		if fromPurl == "" {
			fromPurl = fmt.Sprintf("pkg:generic/%s@%s", fromProtoNode.Name, fromProtoNode.Version)
		}
		fromNode, err := storage.GetNode(nameToNodeID[fromPurl])
		if err != nil {
			return fmt.Errorf("failed to get node: %w", err)
		}
		for _, to := range edge.To {
			toProtoNode := document.GetNodeList().GetNodeByID(to)

			toPurl := string(toProtoNode.Purl())
			if toPurl == "" {
				toPurl = fmt.Sprintf("pkg:generic/%s@%s", toProtoNode.Name, toProtoNode.Version)
			}

			toNode, err := storage.GetNode(nameToNodeID[toPurl])
			if err != nil {
				return fmt.Errorf("failed to get node: %w", err)
			}

			err = fromNode.SetDependency(storage, toNode)
			if errors.Is(err, pkg.ErrSelfDependency) {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to set dependency: %w", err)
			}
		}
	}
	return nil
}
