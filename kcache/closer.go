package kcache

type Closer interface {
	Close() error
}
