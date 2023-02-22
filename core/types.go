package core

import (
	"context"
	"gdut-drcom-go/lib/log"
	"net/netip"
	"time"
)

type Drcom struct {
	Tag            string
	RemoteIP       netip.Addr
	RemotePort     uint16
	KeepAlive1Flag byte
	EnableCrypt    bool
	BindDevice     string
	BindToAddr     bool
	//
	ctx    context.Context
	logger *log.Logger
}

var (
	maxRetry     = 5
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)
