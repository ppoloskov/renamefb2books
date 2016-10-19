package main

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	sqlite "github.com/mattn/go-sqlite3"
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

var ErrNoID = errors.New("No lib.rus.ec id was found")

func (b *Book) GetIDbyMD5(db *sql.DB) error {
	const md5q = `SELECT b.bid, COUNT(b.bid)
		FROM  libbook b
		WHERE b.md5 = ?`

	var LREId, Count sql.NullInt64
	if err := db.QueryRow(md5q, b.MD5).Scan(&LREId, &Count); err != nil {
		return err
	}
	// if LREId > 0 && LREDeleted.Int64 == 1 {
	// 	fmt.Println(LREId)
	// 	const delq = `SELECT libjoinedbooks.GoodId
	// 		FROM libbook
	// 		LEFT JOIN libjoinedbooks ON libjoinedbooks.BadId = libbook.bid
	// 		WHERE libbook.bid = ?
	// 		AND libbook.FileType = 'fb2'`

	// 	if err := db.QueryRow(delq, LREId).Scan(&LREId); err != nil {
	// 		return err
	// 	}
	// }

	if int(LREId.Int64) < 1 {
		return ErrNoID
	}

	b.ID = int(LREId.Int64)
	return nil
}

func (b *Book) GetIDbyTitle(db *sql.DB) error {
	// If not found let's try to find it by name and authors last name
	const q = `SELECT MAX(b.bid) FROM libbook b 
		LEFT JOIN libavtor ON libavtor.bid = b.bid 
		LEFT JOIN libavtors a ON libavtor.aid = a.aid 
		WHERE b.title = ? 
		AND a.LastName = ?
		AND b.Deleted != '1' 
		AND b.FileType = 'fb2'`

	var LREId sql.NullInt64
	if err := db.QueryRow(q, b.Title, b.Authors[0].Lname).Scan(&LREId); err != nil {
		return err
	}

	if LREId.Valid {
		b.ID = int(LREId.Int64)
		return nil
	}
	return ErrNoID
}

func (b *Book) GetTitle(db *sql.DB) error {
	const titleq = `SELECT b.Title
		FROM  libbook b
		WHERE b.bid = ?`

	var title string
	if err := db.QueryRow(titleq, b.ID).Scan(&title); err != nil {
		return err
	}

	b.CorrectTitle = title
	return nil
}

func (b *Book) GetSeries(db *sql.DB) error {
	const GetAuthorsSQL = `SELECT libseq.sid, seqname, sn, libseqs.type 
			FROM 'libseqs'
			INNER JOIN libseq ON libseqs.sid = libseq.sid
			INNER JOIN libbook ON libseq.bid = libbook.bid 		
			WHERE libbook.bid = ? 
			AND libbook.FileType = 'fb2'`

	rows, err := db.Query(GetAuthorsSQL, b.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		s := Sequence{}
		var t string
		err := rows.Scan(&s.LRSId, &s.Name, &s.Number, &t)
		if err != nil {
			return err
		}
		if t == "a" {
			b.CorrectAuthorSequences = append(b.CorrectAuthorSequences, s)
		} else {
			b.CorrectPublisherSequences = append(b.CorrectPublisherSequences, s)
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func (b *Book) GetAuthors(db *sql.DB) error {
	const getauthorsq = `SELECT a.aid, a.FirstName, a.MiddleName, a.LastName, a.NickName, l.srclang
			FROM libbook 
			JOIN libavtor ON libavtor.bid = libbook.bid
			JOIN libavtors a ON a.aid = libavtor.aid
			LEFT JOIN libsrclang AS l ON l.bid = libbook.bid
			WHERE libbook.bid = ?
			AND libbook.FileType = 'fb2'
			AND libavtor.role = 'a'
			ORDER BY title DESC`

	rows, err := db.Query(getauthorsq, b.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		p := Person{}
		var orig sql.NullString

		err := rows.Scan(&p.LRSId, &p.Fname, &p.Mname, &p.Lname, &p.Nick, &orig)
		if err != nil {
			return err
		}

		if orig.Valid {
			if orig.String == "ru" {
				p.Lang = "rus"
			} else {
				p.Lang = "for"
			}
		}
		b.CorrectAuthors = append(b.CorrectAuthors, p)
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func (a *Person) GetPopGenres(db *sql.DB) error {
	const GetGenres = `SELECT code, COUNT(code) FROM libavtors
			INNER JOIN libavtor ON libavtors.aid = libavtor.aid 
			INNER JOIN libbook ON libavtor.bid = libbook.bid 
			INNER JOIN libgenre ON libgenre.bid = libbook.bid 
			INNER JOIN libgenres ON libgenre.gid = libgenres.gid 
			WHERE libavtors.aid = ?
			AND libbook.Deleted != 1
			GROUP BY code
			ORDER BY count(code) DESC
			LIMIT 5
	`

	rows, err := db.Query(GetGenres, a.LRSId)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return err
	}

	for rows.Next() {
		var g string
		var c int
		if err := rows.Scan(&g, &c); err != nil {
			log.Fatal(err)
			return err
		}
		a.Genres = append(a.Genres, g)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
