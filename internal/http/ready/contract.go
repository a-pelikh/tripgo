package ready

import "context"

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type readinessChecker interface {
	Ping(ctx context.Context) error
}
