package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/manduinca/yanakilla-mailseeker/internal/mailparse"
	"github.com/manduinca/yanakilla-mailseeker/internal/zinc"
)

type stats struct {
	indexed atomic.Int64
	failed  atomic.Int64
	bytes   atomic.Int64
}

func main() {
	var (
		zincURL    = flag.String("zinc", envOr("ZINC_URL", "http://localhost:4080"), "URL de ZincSearch")
		zincUser   = flag.String("user", envOr("ZINC_USER", "admin"), "usuario de ZincSearch")
		zincPass   = flag.String("pass", envOr("ZINC_PASSWORD", "Complexpass#123"), "password de ZincSearch")
		index      = flag.String("index", "emails", "nombre del índice")
		batchSize  = flag.Int("batch", 1000, "documentos por request bulk")
		workers    = flag.Int("workers", runtime.NumCPU(), "goroutines de parseo e ingesta")
		cpuProfile = flag.String("cpuprofile", "", "ruta del perfil de CPU")
		memProfile = flag.String("memprofile", "", "ruta del perfil de memoria")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "uso: indexer [flags] <directorio>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	root := flag.Arg(0)

	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		log.Fatalf("no es un directorio accesible: %s", root)
	}

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatalf("cpuprofile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("cpuprofile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	client := zinc.New(*zincURL, *zincUser, *zincPass)
	if err := client.EnsureIndex(*index); err != nil {
		log.Fatalf("no se pudo crear el índice: %v", err)
	}

	start := time.Now()
	st := run(client, *index, root, *workers, *batchSize)
	elapsed := time.Since(start)

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatalf("memprofile: %v", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("memprofile: %v", err)
		}
	}

	report(st, elapsed, *workers, *batchSize)
}

func run(client *zinc.Client, index, root string, workers, batchSize int) *stats {
	st := &stats{}
	paths := make(chan string, workers*batchSize)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consume(client, index, root, paths, batchSize, st)
		}()
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		paths <- path
		return nil
	})
	close(paths)
	wg.Wait()

	if err != nil {
		log.Printf("aviso: el recorrido terminó con error: %v", err)
	}
	return st
}

func consume(client *zinc.Client, index, root string, paths <-chan string, batchSize int, st *stats) {
	batch := make([]any, 0, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := client.Bulk(index, batch); err != nil {
			log.Printf("bulk falló (%d documentos): %v", len(batch), err)
			st.failed.Add(int64(len(batch)))
		} else {
			st.indexed.Add(int64(len(batch)))
		}
		batch = batch[:0]
	}

	for path := range paths {
		email, err := mailparse.ParseFile(path, root)
		if err != nil {
			st.failed.Add(1)
			continue
		}
		st.bytes.Add(int64(len(email.Content)))
		batch = append(batch, email)
		if len(batch) >= batchSize {
			flush()
		}
	}
	flush()
}

func report(st *stats, elapsed time.Duration, workers, batchSize int) {
	indexed := st.indexed.Load()
	failed := st.failed.Load()
	mb := float64(st.bytes.Load()) / (1 << 20)

	rate := 0.0
	if elapsed.Seconds() > 0 {
		rate = float64(indexed) / elapsed.Seconds()
	}

	fmt.Printf("\n")
	fmt.Printf("  workers      %d\n", workers)
	fmt.Printf("  batch        %d\n", batchSize)
	fmt.Printf("  indexados    %d\n", indexed)
	fmt.Printf("  fallidos     %d\n", failed)
	fmt.Printf("  contenido    %.1f MB\n", mb)
	fmt.Printf("  duración     %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  throughput   %.0f docs/s\n\n", rate)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
