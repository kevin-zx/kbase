package kcache

type KCache interface {
	Get(key string) ([]byte, error)
	Save(key string, value []byte) error
	Delete(key string) error
}

type KCloseCache interface {
	KCache
	Closer
}
