package resources

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func BenchmarkCopyStatusNested(b *testing.B) {
	for _, depth := range []int{4, 8, 16} {
		b.Run(fmt.Sprintf("depth-%d", depth), func(b *testing.B) {
			dir := b.TempDir()
			source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
			for _, root := range []string{source, target} {
				for level := 0; level < depth; level++ {
					root = filepath.Join(root, "nested")
					if err := os.MkdirAll(root, 0o755); err != nil {
						b.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(root, "file"), []byte("contents"), 0o644); err != nil {
						b.Fatal(err)
					}
				}
			}
			resource := NewCopy(source, target, false)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				status, err := resource.Status(context.Background())
				if err != nil || status.State != engine.StateSatisfied {
					b.Fatalf("Status = %+v, %v", status, err)
				}
			}
		})
	}
}

func BenchmarkCopyLargeFile(b *testing.B) {
	dir := b.TempDir()
	source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
	const size = 16 * 1024 * 1024
	if err := os.WriteFile(source, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := copyFile(source, target, 0o644); err != nil {
		b.Fatal(err)
	}
	b.Run("compare", func(b *testing.B) {
		b.SetBytes(size)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			matches, err := filesMatch(source, target)
			if err != nil || !matches {
				b.Fatalf("filesMatch = %v, %v", matches, err)
			}
		}
	})
	b.Run("copy", func(b *testing.B) {
		b.SetBytes(size)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := copyFile(source, target, 0o644); err != nil {
				b.Fatal(err)
			}
		}
	})
}
