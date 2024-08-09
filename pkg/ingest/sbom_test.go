package ingest

import (
	"sort"
	"testing"

	"github.com/bit-bom/minefield/pkg"
)

func TestIngestSBOM(t *testing.T) {
	tests := []struct {
		name     string
		sbomPath string
		want     map[uint32]*pkg.Node
		wantErr  bool
	}{
		{
			name:     "default",
			sbomPath: "../../test",
			want: map[uint32]*pkg.Node{
				1: {
					ID:   1,
					Type: "application",
					Name: "pkg:generic/dep1@1.0.0",
				},
				2: {
					ID:   2,
					Type: "library",
					Name: "pkg:generic/dep2@1.0.0",
				},
				3: {
					ID:   3,
					Type: "library",
					Name: "pkg:generic/lib-A@1.0.0",
				},
				4: {
					ID:   4,
					Type: "library",
					Name: "pkg:generic/lib-B@1.0.0",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := pkg.NewMockStorage()
			if err := SBOM(test.sbomPath, storage); test.wantErr != (err != nil) {
				t.Errorf("Sbom() error = %v, wantErr = %v", err, test.wantErr)
			}

			keys, err := storage.GetAllKeys()
			if err != nil {
				t.Fatalf("Failed to get all keys, %v", err)
			}

			sort.Slice(keys, func(i, j int) bool {
				return keys[i] < keys[j]
			})

			for _, key := range keys {
				node, err := storage.GetNode(key)
				if err != nil {
					t.Fatalf("Failed to get node, %v", err)
				}

				if !nodeEquals(node, test.want[key]) {
					t.Fatalf("expected node %v, got %v", test.want[key], node)
				}
			}
		})
	}
}

func nodeEquals(n, n2 *pkg.Node) bool {
	if ((n == nil || n2 == nil) && n != n2) ||
		(n != nil && (n.ID != n2.ID || n.Type != n2.Type)) {
		return false
	}
	return true
}
