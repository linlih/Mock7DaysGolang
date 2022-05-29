package main

import (
	"fmt"
	"geeorm"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	/*db, _ := sql.Open("sqlite3", "gee.db")
	defer func() { _ = db.Close() }()
	_, _ = db.Exec("DROP TABLE IF EXISTS User;")
	_, _ = db.Exec("CREATE TABLE User(Name text);")
	result, err := db.Exec("INSERT INTO User(`Name`) VALUES (?), (?)", "Tom", "Sam")
	if err == nil {
		affected, _ := result.RowsAffected()
		log.Println(affected)
	}
	row := db.QueryRow("SELECT Name FROM User LIMIT 1")
	var name string
	if err := row.Scan(&name); err == nil {
		log.Println(name)
	}*/
	engine, _ := geeorm.NewEngine("sqlite3", "gee.db")
	defer engine.Close()
	s := engine.NewSession()
	_, _ = s.Row("DROP TABLE IF EXISTS User;").Exec()
	_, _ = s.Row("CREATE TABLE User(Name text);").Exec()
	_, _ = s.Row("CREATE TABLE User(Name text);").Exec()
	result, _ := s.Row("INSERT INTO User(`Name`) VALUES (?), (?)", "Tom", "Sam").Exec()
	count, _ := result.RowsAffected()
	fmt.Printf("Exec Success, %d affected\n", count)
}
