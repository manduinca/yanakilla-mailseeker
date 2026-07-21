package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/manduinca/yanakilla-mailseeker/internal/mailparse"
	"github.com/manduinca/yanakilla-mailseeker/internal/zinc"
)

type entry struct {
	folder string
	path   string
	raw    string
}

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
		limit      = flag.Int("limit", 0, "procesar como máximo N correos, 0 para todos")
		extract    = flag.String("extract", "", "reconstruir el arbol de correos en este directorio en vez de indexar")
		cpuProfile = flag.String("cpuprofile", "", "ruta del perfil de CPU")
		memProfile = flag.String("memprofile", "", "ruta del perfil de memoria")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "uso: indexer [flags] <directorio|archivo.csv>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	source := flag.Arg(0)

	info, err := os.Stat(source)
	if err != nil {
		log.Fatalf("no se pudo leer %s: %v", source, err)
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

	if *extract != "" {
		if info.IsDir() {
			log.Fatalf("-extract requiere un archivo csv como origen")
		}
		start := time.Now()
		n, err := extractCSV(source, *extract, *limit)
		if err != nil {
			log.Fatalf("extract: %v", err)
		}
		fmt.Printf("\n  extraidos    %d correos en %s\n  destino      %s\n\n", n, time.Since(start).Round(time.Millisecond), *extract)
		return
	}

	client := zinc.New(*zincURL, *zincUser, *zincPass)
	if err := client.EnsureIndex(*index); err != nil {
		log.Fatalf("no se pudo crear el índice: %v", err)
	}

	start := time.Now()
	st, err := run(client, *index, source, info.IsDir(), *workers, *batchSize, *limit)
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("aviso: la lectura termino con error: %v", err)
	}

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

func run(client *zinc.Client, index, source string, isDir bool, workers, batchSize, limit int) (*stats, error) {
	st := &stats{}
	entries := make(chan entry, workers*64)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consume(client, index, entries, batchSize, st)
		}()
	}

	var err error
	if isDir {
		err = walkDir(source, entries, limit)
	} else {
		err = readCSV(source, entries, limit)
	}
	close(entries)
	wg.Wait()

	return st, err
}

func walkDir(root string, out chan<- entry, limit int) error {
	sent := 0
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		out <- entry{folder: mailparse.FolderOfEntry(filepath.ToSlash(rel)), path: path}
		sent++
		if limit > 0 && sent >= limit {
			return fs.SkipAll
		}
		return nil
	})
}

func readCSV(source string, out chan<- entry, limit int) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.ReuseRecord = true
	r.FieldsPerRecord = 2

	if _, err := r.Read(); err != nil {
		return err
	}

	sent := 0
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		out <- entry{folder: mailparse.FolderOfEntry(rec[0]), raw: rec[1]}
		sent++
		if limit > 0 && sent >= limit {
			return nil
		}
	}
}

func extractCSV(source, dest string, limit int) (int, error) {
	f, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.ReuseRecord = true
	r.FieldsPerRecord = 2

	if _, err := r.Read(); err != nil {
		return 0, err
	}

	dirs := make(map[string]struct{})
	written := 0

	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return written, nil
		}
		if err != nil {
			return written, err
		}

		name := filepath.FromSlash(strings.TrimSuffix(rec[0], "."))
		full := filepath.Join(dest, name)
		dir := filepath.Dir(full)

		if _, ok := dirs[dir]; !ok {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return written, err
			}
			dirs[dir] = struct{}{}
		}

		if err := os.WriteFile(full, []byte(rec[1]), 0o644); err != nil {
			return written, err
		}

		written++
		if limit > 0 && written >= limit {
			return written, nil
		}
	}
}

func consume(client *zinc.Client, index string, entries <-chan entry, batchSize int, st *stats) {
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

	for e := range entries {
		var (
			email *mailparse.Email
			err   error
		)

		if e.path != "" {
			email, err = mailparse.ParseFile(e.path, "")
			if err == nil {
				email.Folder = e.folder
			}
		} else {
			email, err = mailparse.Parse(strings.NewReader(e.raw), e.folder)
		}

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
