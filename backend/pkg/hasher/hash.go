package hash

import "golang.org/x/crypto/bcrypt"

type Bcrypt struct {
	cost int
}

func New(cost int) *Bcrypt {
	if cost == 0 {
		cost = bcrypt.DefaultCost // 10
	}
	return &Bcrypt{cost: cost}
}

func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (b *Bcrypt) Compare(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
