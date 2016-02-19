package main

import (
	"database/sql"
	"log"
	"os"
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

func CreateDB(dbpath string) (db *sql.DB, err error) {

	if dbpath == "" {
		dbpath = "./foo.db"
	}

	os.Remove(dbpath)

	db, err = sql.Open(dbtype, dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(CreateBase)
	if err != nil {
		log.Printf("%q: %s\n", err, CreateBase)
		return nil, err
	} else {
		return db, nil
	}
}

func AddBookDB(db *sql.DB, b *Book) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	res, err := db.Exec("Insert into books(title, filepath) values(?, ?)", b.Title, b.Path)
	if err != nil {
		log.Fatal(err)
	}
	b_id, _ := res.LastInsertId()
	for _, a := range b.Authors {
		res, err = db.Exec("Insert into authors(fname, mname, lname) values(?, ?, ?)", a.Fname, a.Mname, a.Lname)
		if err != nil {
			log.Fatal(err)
		}
		var a_id int
		err = db.QueryRow("select id from authors where fname = ? and mname = ? and lname = ?", a.Fname, a.Mname, a.Lname).Scan(&a_id)
		if err != nil {
			log.Fatal(err)
		}
		res, err = db.Exec("Insert into booksauthors(bookid, authorid) values(?, ?)", b_id, a_id)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, genre := range b.Genres {
		if genre == "" {
			continue
		}
		res, err = db.Exec("Insert into genres(genre) values(?)", genre)
		if err != nil {
			log.Fatal(err)
		}
		var g_id int
		err = db.QueryRow("select id from genres where genre = ?", genre).Scan(&g_id)
		if err != nil {
			log.Fatal(err)
		}
		res, err = db.Exec("Insert into booksgenres(bookid, genreid) values(?, ?)", b_id, g_id)
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, s := range b.Sequences {
		res, err = db.Exec("Insert into sequences(sequence) values(?)", s.Name)
		if err != nil {
			log.Fatal(err)
		}
		var s_id int
		err = db.QueryRow("select id from sequences where sequence = ?", s.Name).Scan(&s_id)
		if err != nil {
			log.Fatal(err)
		}
		res, err = db.Exec("Insert into bookssequence(bookid, sequenceid, seqno) values(?, ?, ?)", b_id, s_id, s.Number)
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()

}
