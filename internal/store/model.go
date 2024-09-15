package store

type User struct {
	ID    int
	Login string
	Hash  []byte
}
