package geeorm

import (
	"database/sql"
	"geeorm/log"
	"geeorm/session"
)

type Engine struct {
	db *sql.DB
}

func NewEngine(drive, source string) (e *Engine, err error) {
	db, err := sql.Open(drive, source)
	if err != nil {
		log.Error(err)
		return
	}
	if err = db.Ping(); err != nil {
		log.Error(err)
		return
	}
	e = &Engine{db: db}
	log.Info("Connect database success")
	return
}

func (e *Engine) Close() {
	if err := e.db.Close(); err != nil {
		log.Error("Fail to close database", err)
		return
	}
	log.Info("Close database success")
}

func (e *Engine) NewSession() *session.Session {
	return session.New(e.db)
}
