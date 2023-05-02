package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	schema = `CREATE TABLE products (
    brand text,
	bslug text,
	category text,
    name text,
	desc text NULL,
    props text NULL,
    image text NULL,
    files text NULL,
    ratings text NULL,
	slug text NULL
);`
	numc = 10
)

//go:generate ./node_modules/.bin/tailwind --minify -i ./src/in.css -o ./assets/css/main.min.css

//go:embed base/* partials/*
var tmplfs embed.FS

//go:embed assets/*
var assets embed.FS

type templateHandler map[string]*template.Template

var (
	tmpls = make(templateHandler)
)

func (th templateHandler) Handler(p string, name string, d any) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		t, ok := th[p+".html"]
		if !ok {
			http.Error(rw, "page not found", http.StatusNotFound)
			return
		}
		err := t.ExecuteTemplate(&buf, name, d)
		if err != nil {
			log.Println(err)
			http.Error(rw, "internal server error", http.StatusInternalServerError)
			return
		}
		buf.WriteTo(rw)
	})
}

type pt interface {
	product | []product
}

type product struct {
	Brand    *string `json:"brand"`
	Bslug    *string `json:"bslug"`
	Category *string `json:"category"`
	Name     *string `json:"name"`
	Desc     *string `json:"desc"`
	Props    *string `json:"props"`
	Image    *string `json:"image"`
	Files    *string `json:"files"`
	Ratings  *string `json:"ratings"`
	Slug     *string `json:"slug"`
}

type contentp struct {
	Title, Desc, Entries string
}

func update(ctx context.Context, db *sqlx.DB, shtsrv *sheets.Service) error {
	log.Println("getting spreadsheet values")
	sht, err := shtsrv.Spreadsheets.Values.Get(os.Getenv("SHEET_ID"), os.Getenv("SHEET_NAME")).Do()
	if err != nil {
		return fmt.Errorf("error getting sheet: %w", err)
	}

	log.Println("starting insert transaction")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning tx: %w", err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM products")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error removing from table: %w", err)
	}
	for i, row := range sht.Values {
		if i == 0 {
			continue
		}
		if len(row) < numc {
			row = append(row, make([]any, numc-len(row))...)
		}
		insert, err := tx.PrepareContext(ctx, "INSERT INTO products (brand, bslug, category, name, desc, props, image, files, ratings, slug) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error preparing insert stmt: %w", err)
		}

		_, err = insert.ExecContext(ctx, row...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting: %w", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error commiting tx: %w", err)
	}
	return nil
}

func maxAgeHandler(seconds int, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", seconds))
		h.ServeHTTP(w, r)
	})
}

func getJson[T pt](ctx context.Context, stmt *sqlx.Stmt, multiple bool, args ...any) (string, error) {
	dst := new(T)
	if multiple {
		err := stmt.SelectContext(ctx, dst, args...)
		if err != nil {
			return "", err
		}
	} else {
		err := stmt.GetContext(ctx, dst, args...)
		if err != nil {
			return "", err
		}
	}
	bs, err := json.Marshal(dst)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func main() {
	log.Println("reading templates")
	bases, err := fs.ReadDir(tmplfs, "base")
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("parsing templates")
	for _, base := range bases {
		if base.IsDir() {
			continue
		}
		t, err := template.ParseFS(tmplfs, path.Join("partials", "*"), path.Join("base", base.Name()))
		if err != nil {
			log.Fatalln(err)
		}
		tmpls[base.Name()] = t
	}
	log.Println("reading location templates")
	locTmpl, err := fs.ReadDir(tmplfs, path.Join("base", "locations"))
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("parsing locations templates")
	for _, loc := range locTmpl {
		t, err := template.ParseFS(tmplfs, path.Join("partials", "*.html"), path.Join("base", "location.html"), path.Join("base", "locations", loc.Name()))
		if err != nil {
			log.Fatalln(err)
		}
		tmpls[loc.Name()] = t
	}
	fmt.Println(tmpls)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db := sqlx.MustOpen("sqlite3", ":memory:")
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, schema)
	if err != nil {
		log.Fatalln("error making table:", err)
	}

	_, err = db.ExecContext(ctx, "CREATE INDEX name_idx ON products(name)")
	if err != nil {
		log.Fatalln("error creating name index:", err)
	}

	shtsrv, err := sheets.NewService(ctx, option.WithAPIKey(os.Getenv("SHEETS_KEY")))
	if err != nil {
		log.Fatalln("error creating sheet service:", err)
	}

	selc, err := db.PreparexContext(ctx, "SELECT * FROM products WHERE category = ?")
	if err != nil {
		log.Fatalln("error preparing select product stmt:", err)
	}
	selb, err := db.PreparexContext(ctx, "SELECT * FROM products WHERE bslug = ?")
	if err != nil {
		log.Fatalln("error preparing select product stmt:", err)
	}
	selprod, err := db.PreparexContext(ctx, "SELECT * FROM products WHERE slug = ? LIMIT 1")
	if err != nil {
		log.Fatalln("error preparing select product stmt:", err)
	}

	products := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vs := mux.Vars(r)
		p, ok := vs["product"]
		if ok {
			j, err := getJson[product](ctx, selprod, false, p)
			if err != nil {
				http.Error(w, "interal server error", http.StatusInternalServerError)
				log.Println(err)
				return
			}
			tmpls.Handler("product", "product", struct {
				Title, Desc, Entry string
			}{
				Entry: j,
			}).ServeHTTP(w, r)
			return
		}
		b, ok := vs["brand"]
		if ok {
			j, err := getJson[[]product](ctx, selb, true, b)
			if err != nil {
				http.Error(w, "interal server error", http.StatusInternalServerError)
				log.Println(err)
				return
			}
			tmpls.Handler("content", "content", contentp{
				Entries: j,
			}).ServeHTTP(w, r)
			return
		}
		c, ok := vs["category"]
		if ok {
			j, err := getJson[[]product](ctx, selc, true, c)
			if err != nil {
				http.Error(w, "interal server error", http.StatusInternalServerError)
				log.Println(err)
				return
			}
			tmpls.Handler("content", "content", contentp{
				Entries: j,
			}).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	r := mux.NewRouter()
	r.NotFoundHandler = tmpls.Handler("404", "404", nil)

	r.PathPrefix("/assets/").Handler(maxAgeHandler(3600, http.StripPrefix("/", http.FileServer(http.FS(assets)))))

	r.Handle("/", tmpls.Handler("index", "index", nil))
	r.Handle("/careers", tmpls.Handler("careers", "careers", nil))
	r.Handle("/about", tmpls.Handler("about", "careers", nil))
	r.Handle("/aquadrive", tmpls.Handler("aquadrive", "careers", nil))
	r.Handle("/affiliates", tmpls.Handler("affiliates", "affiliates", nil))
	r.Handle("/caterpillar", tmpls.Handler("caterpillar", "caterpillar", nil))
	r.Handle("/dockmate", tmpls.Handler("dockmate", "dockmate", nil))
	r.Handle("/electronics", tmpls.Handler("electronics", "electronics", nil))
	r.Handle("/glendinning", tmpls.Handler("glendinning", "glendinning", nil))
	r.Handle("/locations", tmpls.Handler("locations", "locations", nil))

	r.Handle("/locations/north-florida", tmpls.Handler("nfl", "location", nil))
	r.Handle("/locations/south-florida", tmpls.Handler("sfl", "location", nil))
	r.Handle("/locations/central-florida", tmpls.Handler("cfl", "location", nil))
	r.Handle("/locations/virginia", tmpls.Handler("va", "location", nil))
	r.Handle("/locations/michigan", tmpls.Handler("mi", "location", nil))

	productsr := r.PathPrefix("/products/").Subrouter()

	productsr.Handle("/{category}", products)
	productsr.Handle("/{category}/{brand}", products)
	productsr.Handle("/{category}/{brand}/{product}", products)

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err = update(ctx, db, shtsrv)
	if err != nil {
		cancel()
		log.Fatalln("error updating:", err)
	}

	tick := time.NewTicker(time.Hour)

	go func() {
		for {
			select {
			case <-ctx.Done():
				cancel()
				srv.Close()
			case <-tick.C:
				log.Println("updating...")
				err := update(ctx, db, shtsrv)
				if err != nil {
					log.Println("error updating:", err)
				} else {
					log.Println("done updating!")
				}
			}
		}

	}()
	log.Println("listening on 0.0.0.0 8080 updating every 1h")
	log.Fatal(srv.ListenAndServe())
}
