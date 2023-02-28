package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
)

type content struct {
	BrandInfo string
	Pages     []struct {
		Title string `toml:"title"`
		Data  string `toml:"data"`
		Desc  string `toml:"desc"`
	} `toml:"pages"`
}

func build(base, partial, out string) error {
	basefs := os.DirFS(base)
	partialfs := os.DirFS(partial)

	bases, err := fs.ReadDir(basefs, ".")
	if err != nil {
		return err
	}

	partials, err := fs.ReadDir(partialfs, ".")
	if err != nil {
		return err
	}

	var partialnames []string
	for _, f := range partials {
		partialnames = append(partialnames, path.Join(partial, f.Name()))
	}
	bs, err := os.ReadFile("config.toml")
	if err != nil {
		return err
	}

	var c content
	_, err = toml.Decode(string(bs), &c)
	if err != nil {
		return err
	}

	for _, f := range bases {
		fname := f.Name()
		fmt.Println(append(partialnames, path.Join(base, fname)))
		tmpl, err := template.ParseFiles(append(partialnames, path.Join(base, fname))...)
		if err != nil {
			return err
		}

		if fname != "content.html" {
			dir := path.Join(out, strings.ToLower(strings.ReplaceAll(strings.Split(fname, ".")[0], " ", "_")))
			if fname != "index.html" {
				err := os.Mkdir(dir, 0755)
				if err != nil && !errors.Is(err, os.ErrExist) {
					return fmt.Errorf("%s: %w", strings.Split(fname, ".")[0], err)
				}
			}
			var of *os.File
			if fname == "index.html" {
				of, err = os.Create(path.Join(out, "index.html"))
			} else {
				of, err = os.Create(path.Join(dir, "index.html"))
			}
			if err != nil {
				return err
			}

			tmpl.ExecuteTemplate(of, strings.Split(fname, ".")[0], nil)
			of.Close()
			continue
		}

		for _, page := range c.Pages {
			dir := path.Join(out, strings.ToLower(strings.ReplaceAll(page.Title, " ", "_")))
			err := os.Mkdir(dir, 0755)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("%s: %w", page.Title, err)
			}
			p := path.Join(dir, "index.html")
			f, err := os.Create(p)
			if err != nil {
				return fmt.Errorf("%s: %w", page.Title, err)
			}
			d := struct {
				Page struct {
					Title string "toml:\"title\""
					Data  string "toml:\"data\""
					Desc  string "toml:\"desc\""
				}
				Pages content
			}{
				page, c,
			}
			tmpl.ExecuteTemplate(f, "content", d)
			f.Close()
		}
	}
	return nil
}

func main() {
	out := flag.String("out", "", "output dir")
	watch := flag.Bool("w", false, "watch directory")
	flag.Parse()

	if _, err := os.Stat(*out); os.IsNotExist(err) {
		os.Mkdir(*out, 0755)
	}

	if !*watch {
		build("base", "partials", *out)
		return
	}

	log.Println("watching directory...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			case _, ok := <-watcher.Events:
				if !ok {
					done <- true
					return
				}
				start := time.Now()
				err := build("base", "partials", *out)
				if err != nil {
					log.Fatalln(err)
				}
				log.Printf("finished building in %f seconds\n", time.Since(start).Seconds())
			case err, ok := <-watcher.Errors:
				if !ok {
					done <- true
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	dirs := []string{"base", "partials"}

	for _, dir := range dirs {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}
