package main

import (
	"database/sql"
	sqlite "github.com/mattn/go-sqlite3"
	"log"
	"strconv"
	"strings"
)

const (
	CreateBase = `
		create table books (		id integer primary key autoincrement, 
		filepath text,
						title text);

		create table sequences (	id integer primary key autoincrement, 
						sequence text,
						unique (sequence) on conflict ignore);
		create table bookssequence (	bookid integer references books(id),
						sequenceid integer references sequences(id),
						seqno integer);

		create table genres (		id integer primary key autoincrement, 
						genre text,
						unique (genre) on conflict ignore);
		create table booksgenres (	bookid integer references books(id),
						genreid integer references authors(id));
						
		create table authors (		id integer primary key autoincrement, 
						fname text,
						mname text,
						lname text,
						unique (fname, mname, lname) on conflict ignore);
		create table booksauthors (	bookid integer references books(id),
						authorid integer references authors(id));
	`
	dbtype = "sqlite3"
)

func upper(s string) string {
	return strings.ToUpper(s)
}

func OpenDB(dbpath string) (db *sql.DB, err error) {
	if dbpath == "" {
		dbpath = "/Users/p.poloskov/boo.db"
	}

	sql.Register("sqlite3_custom", &sqlite.SQLiteDriver{
		ConnectHook: func(conn *sqlite.SQLiteConn) error {
			if err := conn.RegisterFunc("upp", upper, true); err != nil {
				return err
			}
			return nil
		},
	})

	db, err = sql.Open("sqlite3_custom", dbpath)
	// defer db.Close()

	if err != nil {
		log.Fatal("Failed to create the handle", err)
		return nil, err
	} else {
		return db, err
	}
}

func GetBookID(db *sql.DB, b *Book) (id int) {
	const q = `SELECT MAX(b.bid) FROM libbook b 
		LEFT JOIN libavtor ON libavtor.bid = b.bid 
		LEFT JOIN libavtors a ON libavtor.aid = a.aid 
		WHERE upp(b.title) = upp(?) 
		AND a.LastName = ? 
		AND b.Deleted != '1' 
		AND b.FileType = 'fb2'`

	var LREId string
	err := db.QueryRow(q, b.Title, b.Authors[0].Lname).Scan(&LREId)
	if err != nil {
		log.Println(err)
		// log.Fatal(err)
		return (0)
	}

	id, err = strconv.Atoi(LREId)
	if err != nil {
		return (0)
	}

	return (id)
}

func GetAuthors(db *sql.DB, id int) (Authors []Person) {
	const GetAuthors = `SELECT a.aid, a.FirstName, a.MiddleName, a.LastName FROM libbook 
			LEFT JOIN libavtor ON libavtor.bid = libbook.bid
			LEFT JOIN libavtors a ON a.aid = libavtor.aid
			WHERE libbook.bid = ? 
			AND libbook.Deleted != '1' 
			AND libbook.FileType = 'fb2'
			AND libavtor.role = 'a'
			ORDER BY title DESC`

	rows, err := db.Query(GetAuthors, id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		p := &Person{}
		err := rows.Scan(&p.LRSId, &p.Fname, &p.Mname, &p.Lname)
		if err != nil {
			log.Fatal(err)
		}
		Authors = append(Authors, *p)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	return
}
