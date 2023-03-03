package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	schema = `CREATE TABLE products (
    brand text,
	category text,
    name text,
	desc text NULL,
    props text NULL,
    image text NULL,
    files text NULL,
    ratings text NULL
);`
	numc = 8
)

type product struct {
	Brand, Category, Name, Desc, Props, Image, Files, Ratings sql.NullString
}

func update(ctx context.Context, db *sqlx.DB, shtsrv *sheets.Service) error {
	sht, err := shtsrv.Spreadsheets.Values.Get(os.Getenv("SHEET_ID"), os.Getenv("SHEET_NAME")).Do()
	if err != nil {
		return fmt.Errorf("error getting sheet: %w", err)
	}

	for i, row := range sht.Values {
		if i == 0 {
			continue
		}
		if len(row) < numc {
			row = append(row, make([]any, numc-len(row))...)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("error beginning tx: %w", err)
		}
		insert, err := tx.PrepareContext(ctx, "INSERT INTO products (brand, category, name, desc, props, image, files, ratings) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			return fmt.Errorf("error preparing insert stmt: %w", err)
		}

		_, err = insert.ExecContext(ctx, row...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting: %w", err)
		}

		err = tx.Commit()
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error commiting tx: %w", err)
		}
	}
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db := sqlx.MustOpen("sqlite3", ":memory:")
	err := db.PingContext(ctx)
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

	sel, err := db.PreparexContext(ctx, "SELECT * FROM products")
	if err != nil {
		log.Fatalln("error preparing select stmt:", err)
	}
	selp, err := db.PreparexContext(ctx, "SELECT * FROM products WHERE category = ?")
	if err != nil {
		log.Fatalln("error preparing select product stmt:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	withCtx := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	index := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ps []product
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		err := sel.SelectContext(r.Context(), &ps)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println("error selecting:", err)
			return
		}
		bs, err := json.Marshal(ps)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println("error marshalling:", err)
			return
		}
		w.Write(bs)
	})

	ptype := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ps []product
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")

		v := r.URL.Query()
		cat := v.Get("q")
		if cat == "" {
			http.Error(w, "provide a category", http.StatusBadRequest)
			return
		}

		err := selp.SelectContext(r.Context(), &ps, cat)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println("error selecting:", err)
			return
		}
		bs, err := json.Marshal(ps)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println("error marshalling:", err)
			return
		}
		w.Write(bs)
	})

	http.Handle("/api/all", withCtx(index))
	http.Handle("/api/category", withCtx(ptype))

	srv := &http.Server{
		Addr:         ":" + port,
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

	}()
	log.Println("listening on", port, "updating every 1h")
	log.Fatal(srv.ListenAndServe())
}
